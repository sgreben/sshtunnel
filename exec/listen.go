package sshtunnel

import (
	"context"
	"fmt"
	"net"

	"github.com/sgreben/sshtunnel/connpipe"
)

// Listen is ListenContext with context.Background()
func Listen(laddr net.Addr, raddr string, config *Config) (net.Listener, <-chan error, error) {
	return ListenContext(context.Background(), laddr, raddr, config)
}

// ListenContext serves an SSH tunnel to a remote address on the given local network address `laddr`.
// The remote endpoint of the tunneled connections is given by the network and addr parameters.
func ListenContext(ctx context.Context, laddr net.Addr, raddr string, config *Config) (net.Listener, <-chan error, error) {
	listener, err := net.Listen(laddr.Network(), laddr.String())
	if err != nil {
		return nil, nil, fmt.Errorf("listen on %s://%s: %v", laddr.Network(), laddr.String(), err)
	}
	listenerConnsCh, _ := listenerConns(ctx, listener)
	tunnelConn := func(ctx context.Context) (net.Conn, <-chan error, error) {
		return DialContext(ctx, raddr, config)
	}
	errCh := make(chan error, 1)
	handleListenerConn := func(listenerConn net.Conn) {
		ctxConn, cancel := context.WithCancel(ctx)
		defer listenerConn.Close()
		defer cancel()
		tunnelConn, tunnelConnErrCh, err := tunnelConn(ctxConn)
		if err != nil {
			errCh <- err
			return
		}
		pipeDone := make(chan bool, 1)
		go func() {
			connpipe.WithContext(ctxConn, tunnelConn, listenerConn)
			pipeDone <- true
		}()
		select {
		case <-pipeDone:
			return
		case err := <-tunnelConnErrCh:
			errCh <- err
		case <-ctx.Done():
			errCh <- ctx.Err()
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
			case listenerConn := <-listenerConnsCh:
				go handleListenerConn(listenerConn)
			}
		}
	}()
	return listener, errCh, err
}

func listenerConns(ctx context.Context, listener net.Listener) (<-chan net.Conn, <-chan error) {
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
