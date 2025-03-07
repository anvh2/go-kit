package limiter

import "time"

type TokenBucket struct {
	// tokens is the current number of
	tokens float64
	// capacity is the maximum burst of the bucket.
	capacity float64
	// fillRate is the rate in tokens/second that the bucket will be refilled.
	fillRate float64
	// lastTime is the last time the bucket was updated.
	lastTime int64
}

func NewTokenBucket(capacity, fillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity: capacity,
		fillRate: fillRate,
		lastTime: time.Now().UnixNano(),
	}
}

func (r *TokenBucket) Allow() bool {
	now := time.Now().UnixNano()
	timePassed := now - r.lastTime

	r.tokens += float64(timePassed) * r.fillRate / 1e9
	if r.tokens > r.capacity {
		r.tokens = r.capacity
	}
	r.lastTime = now
	if r.tokens < 1 {
		return false
	}
	r.tokens--
	return true
}
