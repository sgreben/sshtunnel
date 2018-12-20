package sshtunnel

import (
	"net"
	"os/exec"
	"text/template"

	"github.com/sgreben/sshtunnel/backoff"
)

// Config is an SSH tunnel configuration using an external SSH client command
type Config struct {
	// SSH user
	User string
	// SSH server host
	SSHHost string
	// SSH server port
	SSHPort string
	// SSH client command template.
	// A template that may refer to fields from struct commandTemplateData
	// Its output is split according to shell splitting rules and executed.
	CommandTemplate *template.Template
	// This value will be passed to the CommandTemplate in the ExtraArgs field
	CommandExtraArgs string
	// Optional callback to preform any additional configuration of the SSH client command.
	CommandConfig func(*exec.Cmd) error
	// Backoff config used when connecting to the external client.
	Backoff backoff.Config
	// Local IP address to listen on (optional)
	LocalIP *net.IP
}
