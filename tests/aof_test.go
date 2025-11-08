package tests

import (
	"os"
	"strings"
	"testing"

	"github.com/PetarGeorgiev-hash/flashdb/aof"
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
