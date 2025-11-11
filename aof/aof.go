package aof

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PetarGeorgiev-hash/flashdb/protocol"
	"github.com/PetarGeorgiev-hash/flashdb/store"
)

type IAOF interface {
	AppendCommand(args ...string) error
	Reset() error
	Close() error
	LoadAOF(filename string, s store.IStore) error
}

type AOF struct {
	mu   sync.Mutex
	file *os.File
}

func (a *AOF) AppendCommand(args ...string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	cmd := fmt.Sprintf("*%d\r\n", len(args))
	for _, arg := range args {
		cmd += fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg)
	}
	_, err := a.file.WriteString(cmd)
	return err
}

func (a *AOF) Close() error {
	return a.file.Close()
}

func (a *AOF) Reset() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.file.Close(); err != nil {
		return err
	}

	f, err := os.OpenFile(a.file.Name(), os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	a.file = f
	return nil
}

func (a *AOF) LoadAOF(filename string, s store.IStore) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	parser := protocol.NewRESPParser()
	reader := bufio.NewReader(file)

	for {
		parts, err := parser.ParseRESP(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(parts) == 0 {
			continue
		}

		command := strings.ToUpper(parts[0])
		switch command {
		case "SET":
			ttl := time.Duration(0)
			if len(parts) == 4 {
				if sec, err := strconv.Atoi(parts[3]); err == nil {
					ttl = time.Duration(sec) * time.Second
				}
			}
			s.Set(parts[1], []byte(parts[2]), ttl)
		case "DEL":
			s.Delete(parts[1])
		case "EXPIRE":
			key := parts[1]
			sec, _ := strconv.Atoi(parts[2])
			item, _ := s.Get(key)
			if item != nil {
				item.ExpiresAt = time.Now().Add(time.Duration(sec) * time.Second)
			}
		}
	}
	return nil
}

func NewAOF(filename string) (IAOF, error) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &AOF{file: f}, nil
}
