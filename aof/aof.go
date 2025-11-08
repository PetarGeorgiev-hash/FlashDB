package aof

import (
	"fmt"
	"os"
	"sync"
)

type IAOF interface {
	AppendCommand(args ...string) error
	Reset() error
	Close() error
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

func NewAOF(filename string) (IAOF, error) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &AOF{file: f}, nil
}
