package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"syscall"
	"text/template"
)

type SSHDConfig struct {
	Port int
	Username       string
	ListenAddress  string
	AuthorizedKeys string
	LogLevel       string
	HostKey        string
	PidFile        string
}

const sshdConfig = `Protocol 2
AllowUsers {{.Username}} dyno
Port {{.Port}}
ListenAddress {{.ListenAddress}}
AuthorizedKeysFile {{.AuthorizedKeys}}
PasswordAuthentication no
ChallengeResponseAuthentication no
UsePAM no
PermitRootLogin no
LoginGraceTime 20
LogLevel {{.LogLevel}}
PrintLastLog no
HostKey {{.HostKey}}
UsePrivilegeSeparation no
PermitUserEnvironment yes
PidFile {{.PidFile}}
`

var sshdConfigTemplate = template.Must(
	template.New("sshd_config").Parse(sshdConfig))

func writeFile(path string, contents string, perm os.FileMode) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err = file.Chmod(perm); err != nil {
		return err
	}
	if _, err = file.WriteString(contents); err != nil {
		return err
	}

	return nil
}

func writeFileFromTemplate(path string, t *template.Template,
	data interface{}, perm os.FileMode) error {

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err = file.Chmod(perm); err != nil {
		return err
	}
	if err = t.Execute(file, data); err != nil {
		return err
	}

	return nil
}

func fork() (pid int, err syscall.Errno) {
	darwin := runtime.GOOS == "darwin"

	r1, r2, errno := syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)
	if errno != 0 {
		return 0, errno
	}

	if darwin && r2 == 1 {
		r1 = 0
	}

	return int(r1), 0
}

type Rendezvous struct {
	Url string
}

func (r *Rendezvous) connect() {
}

func SpawnSSHD() error {
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

	username, err := exec.Command("whoami").Output()
	if err != nil {
		return fmt.Errorf("whoami error: %s, out: %s", err, username)
	}

	err = writeFile(authorizedKeys, os.Getenv("AUTHORIZED_KEYS"), 0600)
	if err != nil {
		return err
	}
	config := &SSHDConfig{
		Port: 5000,
		Username: strings.TrimSpace(string(username)),
		ListenAddress: "127.0.0.1",
		AuthorizedKeys: authorizedKeys,
		LogLevel: "ERROR",
		HostKey: hostKey,
		PidFile: path.Join(tmp, "sshd.pid"),
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


	hostname, err := exec.Command("hostname").Output()
	if err != nil {
		return fmt.Errorf("hostname error: %s, out: %s", err, hostname)
	}

	url := fmt.Sprintf("rendezvous://rendezvous.runtime.heroku.com:5000/rendezvous-dyno-ssh-ksecret-%s",
		strings.TrimSpace(string(hostname)))
	r := &Rendezvous{url}
	go r.connect()

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
	if err := SpawnSSHD(); err != nil {
		log.Fatal(err)
	}
}
