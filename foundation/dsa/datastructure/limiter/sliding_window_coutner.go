package limiter

import "time"

type SlidingWindowCounter struct {
	windowSize    time.Duration
	maxRequest    int64
	currentWindow int64
	requestCount  int64
	previousCount int64
}

func NewSlidingWindowCounter(windowSize time.Duration, maxRequest int64) *SlidingWindowCounter {
	return &SlidingWindowCounter{
		windowSize:    windowSize,
		maxRequest:    maxRequest,
		currentWindow: 0,
		requestCount:  0,
		previousCount: 0,
	}
}

func (r *SlidingWindowCounter) Allow() bool {
	now := time.Now().UnixNano()
	window := now / r.windowSize.Nanoseconds()
	if window != r.currentWindow {
		r.currentWindow = window
		// reset count and remember the previous count
		r.previousCount = r.requestCount
		r.requestCount = 0
	}
	// calculate the weight of the current window
	windowElasped := float64(now%r.windowSize.Nanoseconds()) / float64(r.windowSize.Nanoseconds())
	currentRequest := int64(windowElasped*float64(r.previousCount)) + r.requestCount
	if currentRequest < r.maxRequest {
		r.requestCount++
		return true
	}
	return false
}
