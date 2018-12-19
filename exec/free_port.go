package sshtunnel

import (
	"fmt"
	"net"
	"strconv"
)

func guessFreePortTCP() (string, int, error) {
	const tcpNet = "tcp"
	listener, err := net.ListenTCP(tcpNet, &net.TCPAddr{})
	if err != nil {
		return "", 0, fmt.Errorf("open temporary listener: %v", err)
	}
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	if err := listener.Close(); err != nil {
		return "", 0, fmt.Errorf("close temporary listener: %v", err)
	}
	portInt64, err := strconv.ParseInt(port, 10, 64)
	if err != nil {
		return port, 0, err
	}
	return port, int(portInt64), nil
}
