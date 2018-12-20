package sshtunnel

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/google/shlex"
)

// CommandTemplateOpenSSHText is a command template text for the openssh `ssh` client binary.
const CommandTemplateOpenSSHText = `ssh -nNT -L "{{.LocalIP}}:{{.LocalPort}}:{{.RemoteAddr}}" -p "{{.SSHPort}}"  "{{.User}}@{{.SSHHost}}" {{.ExtraArgs}}`

// CommandTemplateOpenSSH is a command template for the openssh `ssh` client binary.
var CommandTemplateOpenSSH = mustParse(CommandTemplateOpenSSHText)

// CommandTemplatePuTTYText is a command template text for the PuTTY client.
const CommandTemplatePuTTYText = `putty -ssh -NT "{{.User}}@{{.SSHHost}}" -P "{{.SSHPort}}"  -L "{{.LocalIP}}:{{.LocalPort}}:{{.RemoteAddr}}" {{.ExtraArgs}}`

// CommandTemplatePuTTY is a command template for the PuTTY client.
var CommandTemplatePuTTY = mustParse(CommandTemplatePuTTYText)

type commandTemplateData struct {
	LocalIP    string
	LocalPort  string
	RemoteAddr string
	User       string
	SSHHost    string
	SSHPort    string
	ExtraArgs  string
}

func mustParse(t string) *template.Template {
	return template.Must(template.New("").Parse(t))
}

func commandForTemplate(t *template.Template, data commandTemplateData) (string, []string, error) {
	var buf bytes.Buffer
	err := t.Execute(&buf, data)
	if err != nil {
		return "", nil, fmt.Errorf("execute command template %q: %v", t.Root.String(), err)
	}
	commandText := buf.String()
	var name string
	var args []string
	tokens, err := shlex.Split(commandText)
	if err != nil {
		return "", nil, fmt.Errorf("tokenize command %q: %v", commandText, err)
	}
	if len(tokens) == 0 {
		return "", nil, fmt.Errorf("empty command: %v", commandText)
	}
	name = tokens[0]
	if len(tokens) > 1 {
		args = tokens[1:]
	}
	return name, args, nil
}
