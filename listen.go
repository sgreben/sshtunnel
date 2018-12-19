package sshtunnel

import (
	"context"
	"fmt"
	"net"
)

// Listen is ListenContext with context.Background()
func Listen(laddr net.Addr, network, addr string, config *Config, reconnectBackoff ConfigBackoff) (net.Listener, chan error, error) {
	return ListenContext(context.Background(), laddr, network, addr, config, reconnectBackoff)
}

// ListenContext serves an SSH tunnel to a remote address on the given local network address `laddr`.
// The remote endpoint of the tunneled connections is given by the network and addr parameters.
//
// See func ReDial for a description of the network, addr, config and reconnectBackoff
// parameters.
func ListenContext(ctx context.Context, laddr net.Addr, network, addr string, config *Config, reconnectBackoff ConfigBackoff) (net.Listener, chan error, error) {
	listener, err := net.Listen(laddr.Network(), laddr.String())
	if err != nil {
		return nil, nil, fmt.Errorf("listen on %s://%s: %v", laddr.Network(), laddr.String(), err)
	}
	tunnelConnsCh, tunnelConnsErrCh := ReDialContext(ctx, network, addr, config, reconnectBackoff)
	listenerConnsCh, _ := listenerConns(ctx, listener)
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		defer listener.Close()
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case listenerConn := <-listenerConnsCh:
				defer listenerConn.Close()
				for listenerConn.RemoteAddr() != net.Addr(nil) {
					select {
					case err := <-tunnelConnsErrCh:
						errCh <- err
						return
					case <-ctx.Done():
						errCh <- ctx.Err()
						return
					case tunnelConn := <-tunnelConnsCh:
						connPipeContext(ctx, tunnelConn, listenerConn)
					}
				}
			}
		}
	}()
	return listener, errCh, err
}

func listenerConns(ctx context.Context, listener net.Listener) (<-chan net.Conn, chan error) {
	connCh := make(chan net.Conn)
	errCh := make(chan error)
	go func() {
		defer close(connCh)
		defer close(errCh)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case connCh <- conn:
			}
		}
	}()
	return connCh, errCh
}
