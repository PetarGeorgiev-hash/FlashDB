package protocol

import (
	"bufio"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type Parser interface {
	ReadLine(r *bufio.Reader) (string, error)
	ParseRESP(r *bufio.Reader) ([]string, error)
}

type RESPParser struct {
}

// ParseRESP implements Parser.
func (p *RESPParser) ParseRESP(r *bufio.Reader) ([]string, error) {
	line, err := p.ReadLine(r)
	if err != nil {
		return nil, err
	}

	log.Println(line)
	log.Println(line[0])
	log.Println(len(line))
	if len(line) == 0 || line[0] != '*' {
		return nil, fmt.Errorf("invalid RESP array")
	}

	cmdArrayLenght, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, fmt.Errorf("invalid RESP array length")
	}

	result := make([]string, 0)
	for i := 0; i < cmdArrayLenght; i++ {
		lengthLine, err := p.ReadLine(r)
		if err != nil {
			return nil, err
		}

		if len(lengthLine) == 0 || lengthLine[0] != '$' {
			return nil, fmt.Errorf("expected bulk string, got: %s", lengthLine)
		}

		val, err := p.ReadLine(r)
		if err != nil {
			return nil, err
		}
		result = append(result, val)
	}
	return result, nil
}

// ReadLine implements Parser.
func (*RESPParser) ReadLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSuffix(line, "\r\n")

	return line, nil
}

func NewRESPParser() Parser {
	return &RESPParser{}
}
