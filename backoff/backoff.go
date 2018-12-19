package backoff

import (
	"context"
	"time"
)

// Config is an exponential back-off configuration
// The back-off factor is currently fixed at 2.
type Config struct {
	// Min is the minimum back-off delay (required)
	Min time.Duration
	// Max is the maximum back-off delay (required)
	Max time.Duration
	// MaxAttempts is the maximum total number of attempts (required)
	MaxAttempts int
}

func Run(ctx context.Context, f func() error, config Config) error {
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
