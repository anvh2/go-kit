package limiter

import (
	"time"

	"github.com/anvh2/go-kit/datastructure/queue"
)

type LeakyBucket struct {
	// bucket is the queue that holds the tokens.
	bucket *queue.Queue[struct{}]
	// capacity is the maximum burst of the bucket.
	capacity int64
	// leakyRate is the rate in tokens/second that the bucket will be refilled.
	leakyRate float64
	// lastTime is the last time the bucket was updated.
	lastTime int64
}

func NewLeakyBucket(capacity int64, leakyRate float64) *LeakyBucket {
	return &LeakyBucket{
		bucket:    queue.New[struct{}](capacity),
		capacity:  capacity,
		leakyRate: leakyRate,
		lastTime:  time.Now().UnixNano(),
	}
}

func (r *LeakyBucket) Allow() bool {
	now := time.Now().UnixNano()
	leakyTime := now - r.lastTime
	leaked := int(float64(leakyTime) * r.leakyRate / 1e9)

	if leaked > 0 {
		for i := 0; i < min(leaked, r.bucket.Size()); i++ {
			r.bucket.Dequeue()
		}
		r.lastTime = now
	}

	if r.bucket.Size() < int(r.capacity) {
		r.bucket.Enqueue(struct{}{})
		return true
	}
	return false
}
