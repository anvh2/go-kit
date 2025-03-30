package saga

import (
	"context"
	"time"

	retry "github.com/cenkalti/backoff/v4"
)

const (
	defaultMaxRetries = 5
	defaultBackoff    = 50 * time.Millisecond
)

type Option func(*Options)

type RetryPolicy struct {
	MaxRetries int
	Backoff    time.Duration
}

func (rp RetryPolicy) WithContext(ctx context.Context) retry.BackOffContext {
	backoff := retry.WithMaxRetries(
		retry.NewConstantBackOff(rp.Backoff),
		uint64(rp.MaxRetries),
	)
	retrier := retry.WithContext(backoff, ctx)
	retrier.Reset()
	return retrier
}

type Options struct {
	RetryPolicy RetryPolicy
}

func evaluateOptions(opts []Option) Options {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

func WithDefaultRetryBackoff() Option {
	return WithConstantRetryPolicy(defaultMaxRetries, defaultBackoff)
}

func WithConstantRetryPolicy(maxRetries int, backoff time.Duration) Option {
	if maxRetries < 1 {
		maxRetries = 1
	}
	if backoff < 0 {
		backoff = 0
	}
	return func(o *Options) {
		o.RetryPolicy = RetryPolicy{
			MaxRetries: maxRetries - 1, // 0 based
			Backoff:    backoff,
		}
	}
}
