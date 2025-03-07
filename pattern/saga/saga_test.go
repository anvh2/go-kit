package saga

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSagaExecuteSuccess(t *testing.T) {
	for i := 0; i < 100; i++ {
		wf := New(2)
		counter := 2
		add := Step{
			Transaction: func() error {
				counter++
				return nil
			},
			Compensate: func() error {
				counter--
				return nil
			},
		}
		minus := Step{
			Transaction: func() error {
				counter--
				return nil
			},
			Compensate: func() error {
				counter++
				return nil
			},
		}

		wf.AddStep(context.Background(), add)
		wf.AddStep(context.Background(), minus)

		err := wf.Execute(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, 2, counter)
	}
}

func TestSagaExecuteTransactionFailed(t *testing.T) {
	for i := 0; i < 100; i++ {
		wf := New(2)
		counter := 2
		add := Step{
			Transaction: func() error {
				counter++
				return nil
			},
			Compensate: func() error {
				counter--
				return nil
			},
		}
		minus := Step{
			Transaction: func() error {
				return errors.New("error")
			},
		}

		wf.AddStep(context.Background(), add)
		wf.AddStep(context.Background(), minus)

		err := wf.Execute(context.Background())
		assert.NotNil(t, err)
		assert.Equal(t, 2, counter)
	}
}

func TestSagaExecuteTransactionFailedWithRetry(t *testing.T) {
	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wf := New(2)
			counter := 2
			add := Step{
				Transaction: func() error {
					counter++
					return nil
				},
				Compensate: func() error {
					counter--
					return nil
				},
			}
			minus := Step{
				Transaction: func() error {
					return errors.New("minus-error")
				},
			}

			wf.AddStep(context.Background(), add)
			wf.AddStep(context.Background(), minus)

			err := wf.Execute(context.Background(), WithDefaultRetryBackoff())
			assert.NotNil(t, err)
			assert.Equal(t, 2, counter)
		}()
	}
	wg.Wait()
}

func TestSagaExecuteTransactionFailedAndCompensateFailedWithRetry(t *testing.T) {
	wg := sync.WaitGroup{}
	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wf := New(2)
			counter := 2
			add := Step{
				Transaction: func() error {
					counter++
					return nil
				},
				Compensate: func() error {
					return errors.New("add-compensate-error")
				},
			}
			minus := Step{
				Transaction: func() error {
					return errors.New("minus-error")
				},
			}

			wf.AddStep(context.Background(), add)
			wf.AddStep(context.Background(), minus)

			err := wf.Execute(context.Background(), WithDefaultRetryBackoff())
			assert.NotNil(t, err)
			assert.Equal(t, 3, counter) // compensate can't be executed
		}()
	}
	wg.Wait()
}
