package queue

import "fmt"

type Queue[T any] struct {
	items []T
}

func New[T any](capacity int64) *Queue[T] {
	return &Queue[T]{
		items: make([]T, 0, capacity),
	}
}

func (q *Queue[T]) Enqueue(data T) {
	q.items = append(q.items, data)
}

func (q *Queue[T]) Dequeue() (T, error) {
	if len(q.items) == 0 {
		var zero T
		return zero, fmt.Errorf("queue is empty")
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, nil
}

func (q *Queue[T]) Peek() (T, error) {
	if len(q.items) == 0 {
		var zero T
		return zero, fmt.Errorf("queue is empty")
	}
	return q.items[0], nil
}

func (q *Queue[T]) Size() int {
	return len(q.items)
}

func (q *Queue[T]) IsEmpty() bool {
	return len(q.items) == 0
}
