package replication

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/PetarGeorgiev-hash/flashdb/store"
)

type IManager interface {
	HandleReplicationConn(conn net.Conn)
	fullSync(conn net.Conn) error
	Broadcast(parts []string)
}

type Manager struct {
	mu       sync.Mutex
	replicas map[net.Conn]struct{}
	s        store.IStore
}

func (m *Manager) HandleReplicationConn(conn net.Conn) {
	defer conn.Close()
	log.Printf("[replication] new replica connected from %s", conn.RemoteAddr())

	m.mu.Lock()
	m.replicas[conn] = struct{}{}
	m.mu.Unlock()

	// Step 1: Perform full sync (send snapshot)
	if err := m.fullSync(conn); err != nil {
		log.Printf("[replication] full sync failed: %v", err)
		return
	}

	// Step 2: Keep connection open for async updates
	reader := bufio.NewReader(conn)
	for {
		// Wait for replica ping or disconnect
		_, err := reader.Peek(1)
		if err != nil {
			if err == io.EOF {
				log.Printf("[replication] replica disconnected: %s", conn.RemoteAddr())
			} else {
				log.Printf("[replication] replica read error: %v", err)
			}
			m.mu.Lock()
			delete(m.replicas, conn)
			m.mu.Unlock()
			return
		}
		time.Sleep(5 * time.Second)
	}

}

func (m *Manager) fullSync(conn net.Conn) error {
	data, err := m.s.Export()
	if err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("encode failed: %w", err)
	}

	length := buf.Len()
	conn.Write([]byte(fmt.Sprintf("+FULLSYNC %d\r\n", length)))
	conn.Write(buf.Bytes())
	conn.Write([]byte("+FULLSYNC_END\r\n"))
	return nil
}

func (m *Manager) Broadcast(parts []string) {
	m.mu.Lock()
	if len(m.replicas) == 0 {
		m.mu.Unlock()
		return
	}

	cmd := encodeRESP(parts)
	replicas := make([]net.Conn, 0, len(m.replicas))
	for conn := range m.replicas {
		replicas = append(replicas, conn)
	}
	m.mu.Unlock()

	for _, conn := range replicas {
		go func(c net.Conn) {
			_, err := c.Write([]byte(cmd))
			if err != nil {
				log.Printf("[replication] failed to send to replica %s: %v", c.RemoteAddr(), err)
				m.mu.Lock()
				delete(m.replicas, c)
				m.mu.Unlock()
				c.Close()
			}
		}(conn)
	}
}

func NewManager(s store.IStore) IManager {
	return &Manager{
		replicas: make(map[net.Conn]struct{}),
		s:        s,
	}
}

// TODO: move to protocol/resp.go
func encodeRESP(parts []string) string {
	resp := fmt.Sprintf("*%d\r\n", len(parts))
	for _, p := range parts {
		resp += fmt.Sprintf("$%d\r\n%s\r\n", len(p), p)
	}
	return resp
}
