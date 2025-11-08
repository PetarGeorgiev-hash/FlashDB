package tests

import (
	"os"
	"testing"
	"time"

	"github.com/PetarGeorgiev-hash/flashdb/store"
)

func newTestStore(t *testing.T) store.IStore {
	os.Remove("snapshot.fdb")
	s := store.NewStore()
	t.Cleanup(func() {
		s.Close()
		os.Remove("snapshot.fdb")
	})
	return s
}

func TestSetAndGet(t *testing.T) {
	s := newTestStore(t)
	key := "foo"
	value := []byte("bar")

	_, err := s.Set(key, value, 0)
	if err != nil {
		t.Fatalf("failed to set value: %v", err)
	}

	item, err := s.Get(key)
	if err != nil {
		t.Fatalf("failed to get value: %v", err)
	}

	if string(item.Value) != "bar" {
		t.Errorf("expected 'bar', got '%s'", item.Value)
	}
}

func TestGetExpiredItem(t *testing.T) {
	s := newTestStore(t)
	key := "temp"
	value := []byte("123")

	s.Set(key, value, 1*time.Second)
	time.Sleep(2 * time.Second)

	item, _ := s.Get(key)
	if item != nil {
		t.Errorf("expected nil for expired item, got %v", item)
	}
}
