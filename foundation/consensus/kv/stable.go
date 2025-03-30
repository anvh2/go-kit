package kv

func (m *MemCached) Set(key []byte, val []byte) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.values[string(key)] = val
	return nil
}

// Get returns the value for key, or an empty byte slice if key was not found.
func (m *MemCached) Get(key []byte) ([]byte, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	val, ok := m.values[string(key)]
	if !ok {
		return nil, nil
	}
	return val.([]byte), nil
}

func (m *MemCached) SetUint64(key []byte, val uint64) error {
	return nil
}

// GetUint64 returns the uint64 value for key, or 0 if key was not found.
func (m *MemCached) GetUint64(key []byte) (uint64, error) {
	return 0, nil
}
