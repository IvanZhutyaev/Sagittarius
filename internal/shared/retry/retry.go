package retry

import (
	"context"
	"errors"
	"math"
	"time"
)

type RetryableFunc func() error

func WithExponentialBackoff(ctx context.Context, maxAttempts int, initialDelay time.Duration, fn RetryableFunc) error {
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := time.Duration(float64(initialDelay) * math.Pow(2, float64(attempt-1)))
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry on certain errors
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
	}

	return lastErr
}

