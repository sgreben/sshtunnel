package sshtunnel

import (
	"context"
	"net"

	"github.com/sgreben/sshtunnel/backoff"
)

// ReDial opens a tunnelled connection to the address on the named network.
//
// Failed connections are re-dialled following the given back-off configuration.
// Dropped connections are immediately re-dialed.
//
// Supported networks are "tcp", "tcp4" (IPv4-only), "tcp6" (IPv6-only),
// "unix", "unixgram" and "unixpacket".
func ReDial(network, addr string, config *Config, backoffConfig backoff.Config) (<-chan net.Conn, <-chan error) {
	return ReDialContext(context.Background(), network, addr, config, backoffConfig)
}

// ReDialContext opens a tunnelled connection to the address on the named network using
// the provided context.
//
// Failed connections are re-dialled following the given back-off configuration.
// Dropped connections are immediately re-dialed.
//
// See func ReDial for a description of the network and address
// parameters.
func ReDialContext(ctx context.Context, network, addr string, config *Config, backoffConfig backoff.Config) (<-chan net.Conn, <-chan error) {
	dial := func() (net.Conn, <-chan error, error) {
		return DialContext(ctx, network, addr, config)
	}
	dialBackOff := func() (net.Conn, <-chan error, error) {
		return dialBackOff(ctx, dial, backoffConfig)
	}
	connCh := make(chan net.Conn)
	errCh := make(chan error)
	go func() {
		defer close(connCh)
		defer close(errCh)
		for {
			conn, closedCh, err := dialBackOff()
			if err != nil {
				errCh <- err
				return
			}
			select {
			case connCh <- conn:
			case <-closedCh:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
		}
	}()
	return connCh, errCh
}

func dialBackOff(ctx context.Context, dial func() (net.Conn, <-chan error, error), config backoff.Config) (net.Conn, <-chan error, error) {
	var conn net.Conn
	var connClosedCh <-chan error
	errOut := config.Run(ctx, func() error {
		var err error
		conn, connClosedCh, err = dial()
		return err
	})
	return conn, connClosedCh, errOut
}
