package sshtunnel

import (
	"context"
	"fmt"
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
	var dial DialFunc
	switch network {
	case "tcp", "tcp4", "tcp6":
		dial = DialTCP
	case "unix", "unixgram", "unixpacket":
		dial = DialUnix
	default:
		return nil, nil, fmt.Errorf("unsupported network: %q", network)
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
	conn, err := dial(client, addr)
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

// DialTCP opens a tunneled connection to a remote TCP socket
// The given SSH client must already have an open connection.
var DialTCP DialFunc = func(client *ssh.Client, addr string) (net.Conn, error) {
	host, port, err := splitHostPortInt(addr)
	if err != nil {
		return nil, fmt.Errorf("parse address %q: %v", addr, err)
	}
	msg := struct {
		Host  string
		Port  uint32
		Host0 string
		Port1 uint32
	}{
		Host: host,
		Port: port,
	}
	channelType := "direct-tcpip"
	channel, requests, err := client.OpenChannel(channelType, ssh.Marshal(&msg))
	if err != nil {
		return nil, fmt.Errorf("open %q channel to %s:%d: %v", channelType, host, port, err)
	}
	go ssh.DiscardRequests(requests)
	conn := &channelConn{Channel: channel}
	return conn, nil
}

// DialUnix opens a tunneled connection to a remote unix domain socket
// The given SSH client must already have an open connection.
var DialUnix DialFunc = func(client *ssh.Client, addr string) (net.Conn, error) {
	// See https://github.com/openssh/openssh-portable/blob/master/PROTOCOL
	msg := struct {
		Path      string
		Reserved1 string
		Reserved2 uint32
	}{Path: addr}
	channelType := "direct-streamlocal@openssh.com"
	channel, requests, err := client.OpenChannel(channelType, ssh.Marshal(&msg))
	if err != nil {
		return nil, fmt.Errorf("open %q channel: %v", channelType, err)
	}
	go ssh.DiscardRequests(requests)
	conn := &channelConn{Channel: channel}
	return conn, nil
}
