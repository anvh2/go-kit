package limiter

import "time"

type FixedWindowCounter struct {
	windowSize    time.Duration
	maxRequest    int64
	currentWindow int64
	requestCount  int64
}

func NewFixedWindowCounter(windowSize time.Duration, maxRequest int64) *FixedWindowCounter {
	return &FixedWindowCounter{
		windowSize:    windowSize,
		maxRequest:    maxRequest,
		currentWindow: 0,
		requestCount:  0,
	}
}

func (r *FixedWindowCounter) Allow() bool {
	now := time.Now().UnixNano()
	window := now / r.windowSize.Nanoseconds()
	if window != r.currentWindow {
		r.currentWindow = window
		r.requestCount = 0
	}
	if r.requestCount < r.maxRequest {
		r.requestCount++
		return true
	}
	return false
}
