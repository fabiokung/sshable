package main

import (
	"log"
	"os"
	"os/exec"
	"syscall"
	"text/template"
)

type SSHDConfig struct {
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

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}
	cmd, err := exec.LookPath(os.Args[1])
	if err != nil {
		log.Panicf("Could not find executable %s", os.Args[1])
	}
	syscall.Exec(cmd, os.Args[1:], os.Environ())
}
