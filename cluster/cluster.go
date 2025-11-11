package cluster

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PetarGeorgiev-hash/flashdb/util"
)

// NodeInfo represents a single node in the cluster.
type NodeInfo struct {
	ID       string   `json:"id"`       // Unique ID for node
	Addr     string   `json:"addr"`     // e.g., "127.0.0.1:6379"
	Role     string   `json:"role"`     // "master" or "replica"
	Slots    [2]int   `json:"slots"`    // Start and end slot range owned by this node [1-300]
	Replicas []string `json:"replicas"` // List of replica addresses
}

// Config holds all nodes in the cluster.
type Config struct {
	Nodes []NodeInfo `json:"nodes"`
}

// Manager manages cluster state for the local node.
type Manager struct {
	Self    NodeInfo       // Info about this node
	Nodes   []NodeInfo     // All nodes in the cluster
	SlotMap map[int]string // slot → node address
}

// LoadConfig reads and parses cluster.json into memory.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read cluster config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse cluster config: %w", err)
	}
	return &cfg, nil
}

// NewManager creates a new cluster manager for this node.
func NewManager(cfg *Config, selfAddr string) *Manager {
	m := &Manager{
		Nodes:   cfg.Nodes,
		SlotMap: make(map[int]string),
	}

	// Find myself in the config
	for _, n := range cfg.Nodes {
		if n.Addr == selfAddr {
			m.Self = n
		}
		// Fill slot map
		for slot := n.Slots[0]; slot <= n.Slots[1]; slot++ {
			m.SlotMap[slot] = n.Addr
		}
	}

	return m
}

// GetSlotForKey computes the slot for a given key.
func (m *Manager) GetSlotForKey(key string) int {
	sum := util.CRC16([]byte(key))
	return int(sum % 1024)
}

// IsLocal returns true if this node owns the given slot.
func (m *Manager) IsLocal(slot int) bool {
	return slot >= m.Self.Slots[0] && slot <= m.Self.Slots[1]
}

// GetOwner returns the address of the node that owns the slot.
func (m *Manager) GetOwner(slot int) string {
	if addr, ok := m.SlotMap[slot]; ok {
		return addr
	}
	return ""
}
