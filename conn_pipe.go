package sshtunnel

import (
	"context"
	"io"
	"net"
	"sync"
)

func connPipeContext(ctx context.Context, a net.Conn, b net.Conn) {
	var wg sync.WaitGroup
	ctxAB, cancelAB := context.WithCancel(ctx)
	copyAB := make(chan error)
	go func() {
		_, err := io.Copy(a, b)
		copyAB <- err
	}()
	ctxBA, cancelBA := context.WithCancel(ctx)
	copyBA := make(chan error)
	go func() {
		_, err := io.Copy(b, a)
		copyBA <- err
	}()
	wg.Add(1)
	go func() {
		defer cancelBA()
		defer wg.Done()
		select {
		case <-ctxAB.Done():
		case <-copyAB:
		}
	}()
	wg.Add(1)
	go func() {
		defer cancelAB()
		defer wg.Done()
		select {
		case <-ctxBA.Done():
		case <-copyBA:
		}
	}()
	wg.Wait()
}
