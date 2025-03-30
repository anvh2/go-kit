package kv

import "sync"

type MemCached struct {
	lock   sync.RWMutex
	values map[string]interface{}
}

func NewMemCached() (*MemCached, error) {
	return &MemCached{
		lock:   sync.RWMutex{},
		values: make(map[string]interface{}),
	}, nil
}
