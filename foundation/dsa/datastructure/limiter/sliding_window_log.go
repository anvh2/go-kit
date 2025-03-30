package limiter

import (
	"time"

	"github.com/anvh2/go-kit/foundation/dsa/datastructure/queue"
)

type SlidingWindowLog struct {
	requestLog *queue.Queue[time.Time]
	windowSize time.Duration
	maxRequest int64
}

func NewSlidingWindowLog(windowSize time.Duration, maxRequest int64) *SlidingWindowLog {
	return &SlidingWindowLog{
		requestLog: queue.New[time.Time](maxRequest),
		windowSize: windowSize,
		maxRequest: maxRequest,
	}
}

func (r *SlidingWindowLog) Allow() bool {
	now := time.Now()
	for last, err := r.requestLog.Peek(); err == nil && r.requestLog.Size() > 0 && now.Sub(last) > r.windowSize; {
		r.requestLog.Dequeue()
	}
	if r.requestLog.Size() < int(r.maxRequest) {
		r.requestLog.Enqueue(now)
		return true
	}
	return false
}
