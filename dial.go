package sshtunnel

import (
	"context"
	"net"

	"golang.org/x/crypto/ssh"
)

// DialFunc is a dialler for tunneled connections.
type DialFunc func(*ssh.Client, string) (net.Conn, error)

// Dial opens a tunnelled connection to the address on the named network.
// Supported networks are "tcp", "tcp4" (IPv4-only), "tcp6" (IPv6-only),
// "unix", "unixgram" and "unixpacket".
func Dial(network, addr string, config *Config) (net.Conn, <-chan error, error) {
	return DialContext(context.Background(), network, addr, config)
}

// DialContext opens a tunnelled connection to the address on the named network using
// the provided context.
//
// See func Dial for a description of the network and address parameters.
func DialContext(ctx context.Context, network, addr string, config *Config) (net.Conn, <-chan error, error) {
	if ctx == nil {
		panic("nil context")
	}
	sshAddr := withDefaultPort(config.SSHAddr, "22")
	sshConfig := config.SSHClient
	connectSSH := func(ctx context.Context) (*ssh.Client, chan error, error) {
		return dialSSH(ctx, sshAddr, sshConfig)
	}
	if config.SSHConn != nil {
		connectSSH = func(ctx context.Context) (*ssh.Client, chan error, error) {
			return dialConnSSH(ctx, config.SSHConn, sshAddr, sshConfig)
		}
	}
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}
	client, wait, err := connectSSH(ctx)
	if err != nil {
		return nil, nil, err
	}
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}
	conn, err := client.Dial(network, addr)
	if err != nil {
		return nil, nil, err
	}
	return conn, wait, nil
}

func dialConnSSH(ctx context.Context, conn net.Conn, sshAddr string, sshConfig *ssh.ClientConfig) (*ssh.Client, chan error, error) {
	c, chans, reqs, err := ssh.NewClientConn(conn, sshAddr, sshConfig)
	if err != nil {
		return nil, nil, err
	}
	client := ssh.NewClient(c, chans, reqs)
	wait := make(chan error)
	go func() {
		wait <- client.Wait()
	}()
	go func() {
		<-ctx.Done()
		client.Close()
	}()
	return client, wait, nil
}

func dialSSH(ctx context.Context, sshAddr string, sshConfig *ssh.ClientConfig) (*ssh.Client, chan error, error) {
	client, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		return nil, nil, err
	}
	wait := make(chan error)
	go func() {
		wait <- client.Wait()
	}()
	go func() {
		<-ctx.Done()
		client.Close()
	}()
	return client, wait, nil
}
