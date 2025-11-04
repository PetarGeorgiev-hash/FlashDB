package internal

import (
	"testing"
	"time"
)

/*
   store_test.go

   These tests validate the core functionality of FlashDB's in-memory store:
   - Setting and getting items
   - Deleting items
   - Handling TTL expiration correctly
   - Ensuring concurrency safety (via the Go race detector)
*/

// TestSetAndGet checks that values can be stored and retrieved successfully.
func TestSetAndGet(t *testing.T) {
	store := NewStore()
	defer store.Close()

	key := "username"
	value := []byte("peter")

	// Set a value with no expiration (ttl = 0)
	_, err := store.Set(key, value, 0)
	if err != nil {
		t.Fatalf("failed to set value: %v", err)
	}

	// Retrieve the value
	item, err := store.Get(key)
	if err != nil {
		t.Fatalf("failed to get value: %v", err)
	}

	if item == nil {
		t.Fatalf("expected item, got nil")
	}

	if string(item.Value) != "peter" {
		t.Errorf("expected %q, got %q", "peter", string(item.Value))
	}
}

// TestDelete checks that Delete removes the key from the store.
func TestDelete(t *testing.T) {
	store := NewStore()
	defer store.Close()

	key := "to_delete"
	value := []byte("bye")

	_, _ = store.Set(key, value, 0)
	if err := store.Delete(key); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	item, _ := store.Get(key)
	if item != nil {
		t.Errorf("expected nil after delete, got %+v", item)
	}
}

// TestTTLExpiration ensures items expire after their TTL.
func TestTTLExpiration(t *testing.T) {
	store := NewStore()
	defer store.Close()

	key := "temp"
	value := []byte("data")

	// Set TTL to 100 milliseconds
	_, _ = store.Set(key, value, 100*time.Millisecond)

	// Immediately check that it's still there
	item, _ := store.Get(key)
	if item == nil {
		t.Fatal("item should exist before TTL expires")
	}

	// Wait long enough for it to expire
	time.Sleep(150 * time.Millisecond)

	// Should now return nil
	item, _ = store.Get(key)
	if item != nil {
		t.Error("expected item to be expired, but it still exists")
	}
}

// TestConcurrentAccess simulates multiple goroutines accessing the store.
// Run this with: go test -race
func TestConcurrentAccess(t *testing.T) {
	store := NewStore()
	defer store.Close()

	var done = make(chan bool)

	for i := 0; i < 50; i++ {
		go func(id int) {
			key := "user-" + time.Now().String()
			value := []byte("data")

			// Mix of Set and Get operations
			store.Set(key, value, time.Second)
			store.Get(key)
			store.Delete(key)

			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < 50; i++ {
		<-done
	}
}
