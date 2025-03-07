package limiter

import (
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	limiter := NewTokenBucket(10, 1) // 10 tokens per second

	// simulate 10 requests per second
	for i := 0; i < 10; i++ {
		if limiter.Allow() {
			t.Errorf("unexpected limit at %d", i)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// wait for the bucket to be refilled
	time.Sleep(time.Second)
	if !limiter.Allow() {
		t.Errorf("unexpected limit at 11")
	}
}

func TestLeakyBucket(t *testing.T) {
	limiter := NewLeakyBucket(10, 1)

	for i := 0; i < 10; i++ {
		if !limiter.Allow() {
			t.Errorf("unexpected limit at %d", i)
		}
	}

	if limiter.Allow() {
		t.Errorf("unexpected limit at 11")
	}
}

func TestFixedWindowCounter(t *testing.T) {
	limiter := NewFixedWindowCounter(time.Minute, 10) // 10 requests per second

	// simulate 10 requests per second
	for i := 0; i < 10; i++ {
		if !limiter.Allow() {
			t.Errorf("unexpected limit at %d", i)
		}
		time.Sleep(time.Second)
	}

	if limiter.Allow() {
		t.Errorf("unexpected limit at 11")
	}
}

func TestSlidingWindowLog(t *testing.T) {
	limiter := NewSlidingWindowLog(time.Minute, 10) // 10 requests per second

	// simulate 10 requests per second
	for i := 0; i < 10; i++ {
		if !limiter.Allow() {
			t.Errorf("unexpected limit at %d", i)
		}
		time.Sleep(time.Second)
	}

	if limiter.Allow() {
		t.Errorf("unexpected limit at 11")
	}
}

func TestSlidingWindowCounter(t *testing.T) {
	limiter := NewSlidingWindowCounter(time.Minute, 10) // 10 requests per second

	// simulate 10 requests per second
	for i := 0; i < 10; i++ {
		if !limiter.Allow() {
			t.Errorf("unexpected limit at %d", i)
		}
	}

	// wait for the window to slide
	time.Sleep(10 * time.Second)
	if limiter.Allow() {
		t.Errorf("unexpected limit at 11")
	}
}
