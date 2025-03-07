package delay

import "context"

type IQueue interface {
	Publish(ctx context.Context, data interface{}) error
	Subscribe(ctx context.Context, handler func(ctx context.Context, data interface{}) error) error
	Commit(ctx context.Context, offset interface{}) error
	Close() error
}

type Queue struct {
}
