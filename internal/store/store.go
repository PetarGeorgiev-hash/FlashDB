package internal

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"sync"
	"time"

	"github.com/PetarGeorgiev-hash/flashdb/internal/util"
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
	Save(filename string) error
	Load(filename string) error
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
Save persists the current state of the store to a file.

By iterating over all shards and writing non-expired items to the file in a binary format.

Adding a file version header for compatibility checks during loading.
Capturing the number of items and writing each item's key, value, and expiration timestamp
to the file in a structured manner allowing for efficient loading later.

After writing all items, it flushes the file to ensure data integrity.
*/
func (s *Store) Save(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	file.Write([]byte(util.FileVersion))

	items := []*Item{}
	for _, shard := range s.shards {
		shard.mu.RLock()
		for _, item := range shard.data {
			if !item.IsExpired() {
				items = append(items, item)
			}
		}
		shard.mu.RUnlock()
	}

	binary.Write(file, binary.LittleEndian, uint32(len(items)))
	for _, item := range items {
		keyBytes := []byte(item.Key)
		valBytes := item.Value
		exp := item.ExpiresAt.UnixNano()

		binary.Write(file, binary.LittleEndian, uint32(len(keyBytes)))
		file.Write(keyBytes)
		binary.Write(file, binary.LittleEndian, uint32(len(valBytes)))
		file.Write(valBytes)
		binary.Write(file, binary.LittleEndian, exp)
	}
	return file.Sync()

}

func autoSave(s *Store) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.Save(util.FileName)
		}
	}
}

func (s *Store) Load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	version := make([]byte, len(util.FileVersion))
	_, err = file.Read(version)
	if err != nil {
		return err
	}

	if string(version) != util.FileVersion {
		return fmt.Errorf("incompatible snapshot version")
	}

	var count uint32
	binary.Read(file, binary.LittleEndian, &count)
	for i := 0; i < int(count); i++ {

		var keyLen uint32
		binary.Read(file, binary.LittleEndian, &keyLen)
		keyBytes := make([]byte, keyLen)
		file.Read(keyBytes) // read the key

		var valLen uint32
		binary.Read(file, binary.LittleEndian, &valLen)
		valBytes := make([]byte, valLen)
		file.Read(valBytes) // read the value

		var exp int64
		binary.Read(file, binary.LittleEndian, &exp)
		expiresAt := time.Unix(0, exp)
		if exp > 0 {
			if time.Now().After(expiresAt) {
				continue
			}
			s.Set(string(keyBytes), valBytes, time.Until(expiresAt))
		} else {
			s.Set(string(keyBytes), valBytes, 0)
		}

	}
	return nil
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
		shards: make([]*shard, util.NumShards),
		stop:   make(chan struct{}),
	}
	for i := range store.shards {
		store.shards[i] = &shard{
			data: make(map[string]*Item),
		}
	}
	if err := store.Load(util.FileName); err != nil && !os.IsNotExist(err) {
		log.Printf("No saved snapshots or failed to load them : %v", err)
	}

	go cleanupExpiredItems(store)
	go autoSave(store)
	return store
}

func hashKey(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}
