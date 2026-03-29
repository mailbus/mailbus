package util

import (
	"context"
	"fmt"
	"time"
)

// RetryFunc is a function that can be retried
type RetryFunc func() error

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries    int           // Maximum number of retry attempts
	InitialDelay  time.Duration // Initial delay before first retry
	MaxDelay      time.Duration // Maximum delay between retries
	BackoffFactor float64       // Multiplier for delay after each retry
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    3,
		InitialDelay:  time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}
}

// Retry executes a function with retry logic
func Retry(ctx context.Context, fn RetryFunc, config *RetryConfig) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error
	delay := config.InitialDelay

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			Debug("Retry attempt %d/%d after %v", attempt, config.MaxRetries, delay)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := fn()
		if err == nil {
			if attempt > 0 {
				Debug("Success after %d attempts", attempt+1)
			}
			return nil
		}

		lastErr = err
		Debug("Attempt %d failed: %v", attempt+1, err)

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * config.BackoffFactor)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", config.MaxRetries+1, lastErr)
}

// RetryWithContext executes a function with retry logic and context
func RetryWithContext(ctx context.Context, fn func(context.Context) error, config *RetryConfig) error {
	return Retry(ctx, func() error {
		return fn(ctx)
	}, config)
}
