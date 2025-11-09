package tests

import (
	"testing"

	"github.com/PetarGeorgiev-hash/flashdb/store"
)

func TestSaveAndLoad(t *testing.T) {
	s := newTestStore(t)

	s.Set("k1", []byte("v1"), 0)
	s.Set("k2", []byte("v2"), 0)

	if err := s.Save("snapshot.fdb"); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	s2 := store.NewStore()
	if err := s2.Load("snapshot.fdb"); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	item, _ := s2.Get("k1")
	if string(item.Value) != "v1" {
		t.Errorf("expected 'v1', got '%s'", item.Value)
	}
}
