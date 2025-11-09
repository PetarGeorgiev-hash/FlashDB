package protocol

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
)

type Parser interface {
	ParseRESP(r *bufio.Reader) ([]string, error)
}

type RESPParser struct{}

// ParseRESP implements a proper RESP2 parser.
func (p *RESPParser) ParseRESP(r *bufio.Reader) ([]string, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}

	if len(line) == 0 || line[0] != '*' {
		return nil, fmt.Errorf("invalid RESP array start: %q", line)
	}

	n, err := strconv.Atoi(string(line[1:]))
	if err != nil {
		return nil, fmt.Errorf("invalid RESP array length: %v", err)
	}

	result := make([]string, 0, n)
	for i := 0; i < n; i++ {
		bulkLenLine, err := readLine(r)
		if err != nil {
			return nil, err
		}
		if len(bulkLenLine) == 0 || bulkLenLine[0] != '$' {
			return nil, fmt.Errorf("expected bulk string, got %q", bulkLenLine)
		}

		length, err := strconv.Atoi(string(bulkLenLine[1:]))
		if err != nil {
			return nil, fmt.Errorf("invalid bulk string length: %v", err)
		}

		// Read exactly length + CRLF
		data := make([]byte, length+2)
		if _, err := r.Read(data); err != nil {
			return nil, err
		}
		result = append(result, string(data[:length]))
	}

	return result, nil
}

func readLine(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(line, []byte("\r\n")), nil
}

func NewRESPParser() Parser {
	return &RESPParser{}
}
