package cache

import (
	"container/list"
)

/**
LRU cache:
	- Set Key
	- Get Key
	- Capacity
	- Evict Key
	- Maintain list of recently used keys by moving them to the front
	- Evict the least recently used key when capacity is reached
	- Use a doubly linked list to maintain the order of keys
	- Use a map to store the key-value pairs for O(1) access
	- Use a struct to represent the key-value pair and its position in the list
	- Use a struct to represent the LRU cache with capacity, map, and list
	- Use a constructor function to create a new LRU cache with a given capacity
	- Use a method to set a key-value pair in the cache
	- Use a method to get a value by key from the cache
	- Use a method to evict the least recently used key when capacity is reached
**/

type Value struct {
	value   interface{}
	element *list.Element
}

type LRU struct {
	capacity int32
	values   map[string]*Value
	elements *list.List
}

func NewLRU(capacity int32) *LRU {
	return &LRU{
		capacity: capacity,
		values:   make(map[string]*Value),
		elements: list.New(),
	}
}

func (c *LRU) Set(key string, value interface{}) error {
	exist, ok := c.values[key]
	if ok {
		c.elements.MoveToFront(exist.element)
		c.values[key].value = value
		return nil
	}

	if len(c.values) >= int(c.capacity) {
		front := c.elements.Front()
		c.elements.Remove(front)
		delete(c.values, front.Value.(string))
	}

	ele := c.elements.PushBack(key)
	c.values[key] = &Value{
		value:   value,
		element: ele,
	}
	return nil
}

func (c *LRU) Get(key string) interface{} {
	return c.values[key]
}
