package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
)

const SSHD_PORT = 5000

func connectWithRendezvous() {
	hostname, err := exec.Command("hostname").Output()
	if err != nil {
		log.Fatalf("hostname error: %s, out: %s", err, hostname)
		return
	}

	url := fmt.Sprintf(
		"rendezvous://rendezvous.runtime.heroku.com:5000/rendezvous-dyno-ssh-ksecret-%s",
		strings.TrimSpace(string(hostname)),
	)
	r, err := NewRendezvous(url)
	if err != nil {
		log.Fatal(err)
		return
	}
	r.Connect()
}

func spawnSSHD(username string) error {
	tmp, err := ioutil.TempDir("", "sshd")
	if err != nil {
		return err
	}

	hostKey := path.Join(tmp, "id_host_rsa")
	authorizedKeys := path.Join(tmp, "authorized_keys")
	sshdConfig := path.Join(tmp, "sshd_config")

	out, err := exec.Command("ssh-keygen", "-t", "rsa", "-q", "-N", "", "-f",
		hostKey).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh-keygen error: %s, out: %s", err, out)
	}
	err = writeFile(authorizedKeys, os.Getenv("AUTHORIZED_KEYS"), 0600)
	if err != nil {
		return err
	}

	config := &SSHDConfig{
		Port:           SSHD_PORT,
		Username:       strings.TrimSpace(username),
		ListenAddress:  "127.0.0.1",
		AuthorizedKeys: authorizedKeys,
		LogLevel:       "ERROR",
		HostKey:        hostKey,
		PidFile:        path.Join(tmp, "sshd.pid"),
	}
	err = writeFileFromTemplate(sshdConfig, sshdConfigTemplate, config, 0600)
	if err != nil {
		return err
	}

	logFile, err := os.Create(path.Join(tmp, "sshd.log"))
	if err != nil {
		return err
	}
	sshd := exec.Command("/usr/sbin/sshd", "-D", "-e", "-f", sshdConfig)
	sshd.Stdout = logFile
	sshd.Stderr = logFile
	if err = sshd.Start(); err != nil {
		return err
	}

	go connectWithRendezvous()
	return sshd.Wait()
}

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}

	pid, errno := fork()
	if errno != 0 {
		log.Panicf("fork error: %s", errno.Error())
	}

	if pid != 0 {
		// parent

		cmd, err := exec.LookPath(os.Args[1])
		if err != nil {
			log.Panicf("Could not find executable %s",
				os.Args[1])
		}
		syscall.Exec(cmd, os.Args[1:], os.Environ())
	}

	// child

	username, err := exec.Command("whoami").Output()
	if err != nil {
		log.Panicf("whoami error: %s, out: %s", err, username)
	}

	if err := spawnSSHD(string(username)); err != nil {
		log.Fatal(err)
	}
}
