package internal

import (
	"fmt"
	"hash/fnv"
	"sync"
	"time"
)

const numShards = 16

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
	Close()
}

type shard struct {
	data map[string]*Item
	mu   sync.RWMutex
}

/*
Store is an in-memory implementation of the IStore interface.

Consists of multiple shards to reduce lock contention and improve concurrency.
Each shard is a separate map with its own mutex for thread-safe access.
Including a background goroutine to periodically clean up expired items.
And a stop channel to signal the cleanup goroutine to stop when the store is closed.
*/
type Store struct {
	shards []*shard
	stop   chan struct{}
}

func (s *Store) Close() {
	close(s.stop)
}

func (s *Store) Delete(key string) error {
	index := s.GetShardIndex(key)
	shard := s.shards[index]
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if _, exists := shard.data[key]; !exists {
		return fmt.Errorf("key not found")
	}
	delete(shard.data, key)
	return nil
}

func (s *Store) Get(key string) (*Item, error) {
	index := s.GetShardIndex(key)
	shard := s.shards[index]
	shard.mu.RLock()
	item, exists := shard.data[key]
	if !exists {
		shard.mu.RUnlock()
		return nil, nil
	}
	if item.IsExpired() {
		shard.mu.RUnlock()
		shard.mu.Lock()
		delete(shard.data, key)
		shard.mu.Unlock()
		return nil, nil
	}
	shard.mu.RUnlock()
	return item, nil
}

func (s *Store) Set(key string, value []byte, ttl time.Duration) (*Item, error) {
	index := s.GetShardIndex(key)
	shard := s.shards[index]

	shard.mu.Lock()
	defer shard.mu.Unlock()
	item := &Item{
		Key:   key,
		Value: value,
	}
	if ttl > 0 {
		item.ExpiresAt = time.Now().Add(ttl)
	}
	shard.data[key] = item
	return item, nil
}

/*
cleanupExpiredItems runs in a background goroutine to periodically remove expired items from the store.

ticker triggers every minute to check each shard for expired items.
and waits for a stop signal to terminate the goroutine when the store is closed.
*/
func cleanupExpiredItems(s *Store) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			for _, shard := range s.shards {
				shard.mu.Lock()
				for key, item := range shard.data {
					if item.IsExpired() {
						delete(shard.data, key)
					}
				}
				shard.mu.Unlock()
			}
		}
	}
}

func (s *Store) GetShardIndex(key string) int {
	return int(hashKey(key) % uint32(len(s.shards)))
}

func NewStore() IStore {
	store := &Store{
		shards: make([]*shard, numShards),
		stop:   make(chan struct{}),
	}
	for i := range store.shards {
		store.shards[i] = &shard{
			data: make(map[string]*Item),
		}
	}

	go cleanupExpiredItems(store)
	return store
}

func hashKey(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}
