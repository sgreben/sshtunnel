package sshtunnel

import (
	"context"
	"fmt"
	"net"

	"github.com/sgreben/sshtunnel/backoff"
	"github.com/sgreben/sshtunnel/connpipe"
)

// Listen is ListenContext with context.Background()
func Listen(laddr net.Addr, network, addr string, config *Config, reconnectBackoff backoff.Config) (net.Listener, chan error, error) {
	return ListenContext(context.Background(), laddr, network, addr, config, reconnectBackoff)
}

// ListenContext serves an SSH tunnel to a remote address on the given local network address `laddr`.
// The remote endpoint of the tunneled connections is given by the network and addr parameters.
//
// See func ReDial for a description of the network, addr, config and reconnectBackoff
// parameters.
func ListenContext(ctx context.Context, laddr net.Addr, network, addr string, config *Config, reconnectBackoff backoff.Config) (net.Listener, chan error, error) {
	listener, err := net.Listen(laddr.Network(), laddr.String())
	if err != nil {
		return nil, nil, fmt.Errorf("listen on %s://%s: %v", laddr.Network(), laddr.String(), err)
	}
	tunnelConnsCh, tunnelConnsErrCh := ReDialContext(ctx, network, addr, config, reconnectBackoff)
	listenerConnsCh, _ := listenerConns(ctx, listener)
	errCh := make(chan error, 1)
	handleListenerConn := func(listenerConn net.Conn) {
		ctxConn, cancel := context.WithCancel(ctx)
		defer listenerConn.Close()
		defer cancel()
		for listenerConn.RemoteAddr() != net.Addr(nil) {
			select {
			case err := <-tunnelConnsErrCh:
				errCh <- err
				return
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case tunnelConn, ok := <-tunnelConnsCh:
				if !ok {
					return
				}
				connpipe.WithContext(ctxConn, tunnelConn, listenerConn)
			}
		}
	}
	go func() {
		defer listener.Close()
		defer close(errCh)
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case listenerConn, ok := <-listenerConnsCh:
				if !ok {
					return
				}
				go handleListenerConn(listenerConn)
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
