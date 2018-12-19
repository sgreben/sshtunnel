package sshtunnel

import (
	"net"
	"strconv"
)

func withDefaultPort(addr, port string) string {
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	return addr + ":" + port
}

func splitHostPortInt(addr string) (string, uint32, error) {
	host, portString, errSplit := net.SplitHostPort(addr)
	port, errPort := strconv.ParseUint(portString, 10, 32)
	if errSplit != nil {
		return host, uint32(port), errSplit
	}
	if errPort != nil {
		return host, uint32(port), errPort
	}
	return host, uint32(port), nil
}
