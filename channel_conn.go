package sshtunnel

import (
	"errors"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

type channelConn struct {
	ssh.Channel
	laddr, raddr net.TCPAddr
}

func (t *channelConn) LocalAddr() net.Addr {
	return &t.laddr
}

func (t *channelConn) RemoteAddr() net.Addr {
	return &t.raddr
}

func (t *channelConn) SetDeadline(deadline time.Time) error {
	if err := t.SetReadDeadline(deadline); err != nil {
		return err
	}
	return t.SetWriteDeadline(deadline)
}

func (t *channelConn) SetReadDeadline(deadline time.Time) error {
	return errors.New("ssh: channelConn: deadline not supported")
}

func (t *channelConn) SetWriteDeadline(deadline time.Time) error {
	return errors.New("ssh: channelConn: deadline not supported")
}
