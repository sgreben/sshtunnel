package sshtunnel

import (
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Config is an SSH tunnel configuration
//
// When `SSHConfig` is set, only the `SSHAddr` field in this struct is required.
// When `SSHConn` is set to a non-nil net.Conn, that connection is reused instead of opening a new one.
type Config struct {
	// SSHAddr is the host:port address of the SSH server (required)
	SSHAddr string
	// Auth contains the authentication settings (required if SSHConfig not set)
	Auth ConfigAuth
	// SSHConn is a pre-existing connection to an SSH server (optional)
	SSHConn net.Conn
	// SSHConfig is a raw ssh.Client config
	SSHConfig *ssh.ClientConfig
}

// ConfigBackoff is an exponential back-off configuration
// The back-off factor is currently fixed at 2.
type ConfigBackoff struct {
	// Min is the minimum back-off delay (required)
	Min time.Duration
	// Max is the maximum back-off delay (required)
	Max time.Duration
	// MaxAttempts is the maximum total number of attempts (required)
	MaxAttempts int
}

// ConfigAuth is the authentication configuration for an SSH tunnel
type ConfigAuth struct {
	UserName string
	Password *string
	SSHAgent *ConfigAuthSSHAgent
	Keys     []ConfigAuthKey
}

// ConfigAuthSSHAgent is the configuration for an ssh-agent connection
type ConfigAuthSSHAgent struct {
	Addr       net.Addr
	Passphrase *[]byte
}

// ConfigAuthKey is the configuration of an ssh key
// Either Signer, or one of PEM and Path must be set.
// If PEM or Path are set and the referred key is encrypted, Passphrase must also be set.
type ConfigAuthKey struct {
	PEM        *[]byte
	Path       *string
	Passphrase *[]byte
	Signer     ssh.Signer
}

// Methods returns the configured SSH auth methods
func (a ConfigAuth) Methods() (out []ssh.AuthMethod, err error) {
	if a.Password != nil {
		out = append(out, ssh.Password(*a.Password))
	}
	var keys []ssh.Signer
	if a.SSHAgent != nil {
		agentKeys, err := a.SSHAgent.Keys()
		if err != nil {
			return nil, err
		}
		keys = append(keys, agentKeys...)
	}
	if a.Keys != nil {
		for _, k := range a.Keys {
			key, err := k.Key()
			if err != nil {
				return nil, err
			}
			keys = append(keys, key)
		}
	}
	if len(keys) > 0 {
		out = append(out, ssh.PublicKeys(keys...))
	}
	return
}

// Keys obtains and returns all keys from the configured ssh agent
func (a ConfigAuthSSHAgent) Keys() ([]ssh.Signer, error) {
	conn, err := net.Dial(a.Addr.Network(), a.Addr.String())
	if err != nil {
		return nil, err
	}
	sshAgent := agent.NewClient(conn)
	signers, err := sshAgent.Signers()
	if err == nil {
		return signers, nil
	}
	if a.Passphrase == nil {
		return nil, err
	}
	if err := sshAgent.Unlock(*a.Passphrase); err != nil {
		return nil, err
	}
	return sshAgent.Signers()
}

// Key obtains and returns the configured key
func (a ConfigAuthKey) Key() (ssh.Signer, error) {
	switch {
	case a.Signer != nil:
		return a.Signer, nil
	case a.PEM != nil && a.Passphrase != nil:
		return ssh.ParsePrivateKeyWithPassphrase(*a.PEM, *a.Passphrase)
	case a.PEM != nil:
		return ssh.ParsePrivateKey(*a.PEM)
	case a.Path != nil && a.Passphrase != nil:
		buf, err := ioutil.ReadFile(*a.Path)
		if err != nil {
			return nil, err
		}
		return ssh.ParsePrivateKeyWithPassphrase(buf, *a.Passphrase)
	case a.Path != nil:
		buf, err := ioutil.ReadFile(*a.Path)
		if err != nil {
			return nil, err
		}
		return ssh.ParsePrivateKey(buf)
	default:
		return nil, fmt.Errorf("no ssh key defined")
	}
}
