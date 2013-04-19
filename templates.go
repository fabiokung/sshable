package main

import (
	"os"
	"text/template"
)

type SSHDConfig struct {
	Port           int
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

func writeFileFromTemplate(path string, t *template.Template, data interface{},
	perm os.FileMode) error {

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
