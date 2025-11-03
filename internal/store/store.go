package internal

import (
	"sync"
	"time"
)

/*
	The Item struct represents a key-value pair with an optional time-to-live (TTL) duration.

If the TTL is set, the item will expire after the specified duration.
If not set, the item will persist indefinitely.
*/
type Item struct {
	Key       string
	Value     []byte
	ExpiresAt time.Time
}

func (i *Item) IsExpired() bool {
	if i.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(i.ExpiresAt)
}

// IStore defines the interface for a key-value store with methods to set, get, delete items, and close the store.
type IStore interface {
	Set(key string, value []byte, ttl time.Duration) (*Item, error)
	Get(key string) (*Item, error)
	Delete(key string) error
	Close() error
}

type Store struct {
	data map[string]*Item
	mu   sync.RWMutex
}

// Close implements IStore.
func (s *Store) Close() error {
	panic("unimplemented")
}

// Delete implements IStore.
func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

// Get implements IStore.
func (s *Store) Get(key string) (*Item, error) {
	s.mu.RLock()
	item, exists := s.data[key]
	if !exists {
		return nil, nil
	}
	if item.IsExpired() {
		s.mu.RUnlock()
		s.mu.Lock()
		delete(s.data, key)
		s.mu.Unlock()
		return nil, nil
	}
	s.mu.RUnlock()
	return item, nil
}
// Set implements IStore.
func (s *Store) Set(key string, value []byte, ttl time.Duration) (*Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := &Item{
		Key:   key,
		Value: value,
	}
	if ttl > 0 {
		item.ExpiresAt = time.Now().Add(ttl)
	}
	s.data[item.Key] = item
	return item, nil
}

func NewStore() IStore {
	return &Store{
		data: make(map[string]*Item, 1),
	}
}


