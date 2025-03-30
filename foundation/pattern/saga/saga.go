package saga

import (
	"context"

	"github.com/anvh2/go-kit/builtin/errors"
	retry "github.com/cenkalti/backoff/v4"
)

var _ ISaga = &Saga{}

type ISaga interface {
	AddStep(ctx context.Context, step Step)
	Execute(ctx context.Context, opts ...Option) error
}

type Transaction func() error
type Compensating func() error

type Saga struct {
	steps []Step
}

type Step struct {
	Transaction Transaction
	Compensate  Compensating
}

func New(size int) *Saga {
	return &Saga{
		steps: make([]Step, 0, size),
	}
}

func (s *Saga) AddStep(ctx context.Context, step Step) {
	s.steps = append(s.steps, step)
}

// Execute executes the Saga step by step
// If transaction in any step fails despite retrying to max attempts, it will rollback entire compensates from the failed transaction back
// If compensate in any step fails despite retrying to max attempts, it will continue to rollback the next compensates
// It will return the last error that occurs during the Saga execution
// Retry policy can be set using the options. If not set, it will not retry
func (s *Saga) Execute(ctx context.Context, opts ...Option) error {
	opt := evaluateOptions(opts)
	var errs errors.Errors
	for currentIdx, currentStep := range s.steps {
		if currentStep.Transaction == nil {
			continue
		}
		// do transaction
		if err := retry.Retry(retry.Operation(currentStep.Transaction), opt.RetryPolicy.WithContext(ctx)); err != nil {
			errs = append(errs, err)
			// rollback compensates from the failed transaction back
			for lookbackId := currentIdx - 1; lookbackId >= 0; lookbackId-- {
				lookbackStep := s.steps[lookbackId]
				if lookbackStep.Compensate == nil {
					continue
				}
				// do compensate
				if err = retry.Retry(retry.Operation(lookbackStep.Compensate), opt.RetryPolicy.WithContext(ctx)); err != nil {
					errs = append(errs, err)
				}
			}
			return errs.Final()
		}
	}
	return errs.Final()
}
