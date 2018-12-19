package sshtunnel

import (
	"context"
	"net"
	"time"
)

// ReDial opens a tunnelled connection to the address on the named network.
//
// Failed connections are re-dialled following the given back-off configuration.
// Dropped connections are immediately re-dialed.
//
// Supported networks are "tcp", "tcp4" (IPv4-only), "tcp6" (IPv6-only),
// "unix", "unixgram" and "unixpacket".
func ReDial(network, addr string, config *Config, backoffConfig ConfigBackoff) (<-chan net.Conn, <-chan error) {
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
func ReDialContext(ctx context.Context, network, addr string, config *Config, backoffConfig ConfigBackoff) (<-chan net.Conn, <-chan error) {
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
			connCh <- conn
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case <-closedCh:
			}
		}
	}()
	return connCh, errCh
}

func dialBackOff(ctx context.Context, dial func() (net.Conn, <-chan error, error), config ConfigBackoff) (net.Conn, <-chan error, error) {
	var conn net.Conn
	var connClosedCh <-chan error
	errOut := backOff(ctx, func() error {
		var err error
		conn, connClosedCh, err = dial()
		return err
	}, config)
	return conn, connClosedCh, errOut
}

func backOff(ctx context.Context, f func() error, config ConfigBackoff) error {
	const backOffFactor = 2
	delay := config.Min
	for i := 1; true; i++ {
		err := f()
		if err == nil {
			return nil
		}
		if i > config.MaxAttempts {
			return err
		}
		delay *= backOffFactor
		if delay > config.Max {
			delay = config.Max
		}
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
