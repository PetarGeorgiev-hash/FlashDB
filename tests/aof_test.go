package tests

import (
	"os"
	"strings"
	"testing"

	"github.com/PetarGeorgiev-hash/flashdb/aof"
	"github.com/PetarGeorgiev-hash/flashdb/store"
	"github.com/PetarGeorgiev-hash/flashdb/util"
)

func TestAppendAndResetAOF(t *testing.T) {
	os.Remove("test.aof")
	a, err := aof.NewAOF("test.aof")
	if err != nil {
		t.Fatalf("failed to create AOF: %v", err)
	}
	defer os.Remove("test.aof")

	a.AppendCommand("SET", "foo", "bar")
	a.AppendCommand("DEL", "foo")

	data, err := os.ReadFile("test.aof")
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if !strings.Contains(string(data), "SET") {
		t.Error("expected SET command in AOF file")
	}

	if err := a.Reset(); err != nil {
		t.Fatalf("failed to reset AOF: %v", err)
	}

	data, _ = os.ReadFile("test.aof")
	if len(data) != 0 {
		t.Error("expected empty file after reset")
	}
}

func TestAOFReplay(t *testing.T) {
	s := store.NewStore()
	a, _ := aof.NewAOF(util.AppendFile)
	defer os.Remove(util.AppendFile)

	// Write commands
	s.Set("foo", []byte("bar"), 0)
	a.AppendCommand("SET", "foo", "bar")

	// Simulate restart
	s2 := store.NewStore()
	err := a.LoadAOF(util.AppendFile, s2)
	if err != nil {
		t.Fatal(err)
	}

	item, _ := s2.Get("foo")
	if string(item.Value) != "bar" {
		t.Fatalf("expected bar, got %s", item.Value)
	}
}
