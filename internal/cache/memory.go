package cache

import "sync"

type MemoryKvStore struct {
	Items map[string]interface{}
	mu    sync.RWMutex
}

func NewMemoryCache() Cache {
	return &MemoryKvStore{Items: make(map[string]interface{})}
}

func (c *MemoryKvStore) Store(key string, i interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Items[key] = i
}

func (c *MemoryKvStore) Get(key string) interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Items[key]
}
