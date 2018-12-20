package sshtunnel

import (
	"context"
	"fmt"
	"net"
	"os/exec"

	"github.com/sgreben/sshtunnel/backoff"
)

// Dial opens a tunnelled connection to the given address using the configured
// external SSH client.
func Dial(remoteAddr string, config *Config) (net.Conn, <-chan error, error) {
	return DialContext(context.Background(), remoteAddr, config)

}

// DialContext opens a tunnelled connection to the given address using the configured
// external SSH client and the provided context.
func DialContext(ctx context.Context, remoteAddr string, config *Config) (net.Conn, <-chan error, error) {
	var localIP net.IP
	if config.LocalIP != nil {
		localIP = *config.LocalIP
	} else {
		localIP = net.ParseIP("127.0.0.1")
	}
	portString, port, err := guessFreePortTCP(localIP)
	if err != nil {
		return nil, nil, err
	}
	dial := func() (net.Conn, error) {
		return net.DialTCP("tcp", nil, &net.TCPAddr{IP: localIP, Port: port})
	}
	name, args, err := commandForTemplate(config.CommandTemplate, commandTemplateData{
		LocalIP:    localIP.String(),
		LocalPort:  portString,
		User:       config.User,
		SSHHost:    config.SSHHost,
		SSHPort:    config.SSHPort,
		RemoteAddr: remoteAddr,
	})
	if err != nil {
		return nil, nil, err
	}

	ctxCmd, cancelCmd := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctxCmd, name, args...)
	if config.CommandConfig != nil {
		if err := config.CommandConfig(cmd); err != nil {
			cancelCmd()
			return nil, nil, err
		}
	}
	if err := cmd.Start(); err != nil {
		cancelCmd()
		return nil, nil, fmt.Errorf("exec: %v", err)
	}

	cmdErrCh := make(chan error, 1)
	errCh := make(chan error, 1)
	go func() { cmdErrCh <- cmd.Wait() }()
	go func() {
		defer cancelCmd()
		select {
		case err := <-cmdErrCh:
			errCh <- err
		case <-ctx.Done():
			errCh <- ctx.Err()
		}
	}()

	connCh := make(chan net.Conn)
	go func() {
		conn, err := dialBackOff(ctx, dial, config.Backoff)
		if err != nil {
			cancelCmd()
			errCh <- err
			return
		}
		connCh <- conn
	}()
	select {
	case conn := <-connCh:
		return conn, errCh, nil
	case err := <-errCh:
		return nil, nil, err
	}
}

func dialBackOff(ctx context.Context, dial func() (net.Conn, error), config backoff.Config) (net.Conn, error) {
	var conn net.Conn
	errOut := config.Run(ctx, func() error {
		var err error
		conn, err = dial()
		return err
	})
	return conn, errOut
}
