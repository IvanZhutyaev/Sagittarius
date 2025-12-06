package circuitbreaker

import (
	"context"
	"errors"
	"time"

	"github.com/sony/gobreaker"
)

type CircuitBreaker struct {
	cb *gobreaker.CircuitBreaker
}

func New(name string, maxRequests uint32, timeout time.Duration) *CircuitBreaker {
	settings := gobreaker.Settings{
		Name:          name,
		MaxRequests:   maxRequests,
		Interval:      time.Minute,
		Timeout:       timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			// Log state changes in production
		},
	}

	return &CircuitBreaker{
		cb: gobreaker.NewCircuitBreaker(settings),
	}
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	result, err := cb.cb.Execute(func() (interface{}, error) {
		return fn()
	})

	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			return nil, ErrCircuitOpen
		}
		if errors.Is(err, gobreaker.ErrTooManyRequests) {
			return nil, ErrTooManyRequests
		}
	}

	return result, err
}

var (
	ErrCircuitOpen    = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests")
)

