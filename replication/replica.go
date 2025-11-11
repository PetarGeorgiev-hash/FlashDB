package replication

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/PetarGeorgiev-hash/flashdb/protocol"
	"github.com/PetarGeorgiev-hash/flashdb/store"
)

func StartReplica(masterAddr string, s store.IStore) error {
	conn, err := net.Dial("tcp", masterAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Printf("[replica] connected to master %s", masterAddr)

	// Step 1: Ask for full sync
	conn.Write([]byte("*1\r\n$4\r\nSYNC\r\n"))

	reader := bufio.NewReader(conn)

	line, _ := reader.ReadString('\n')
	if strings.HasPrefix(line, "+FULLSYNC") {
		parts := strings.Split(strings.TrimSpace(line), " ")
		size := 0
		if len(parts) == 2 {
			size, _ = strconv.Atoi(parts[1])
		}
		log.Printf("[replica] receiving full sync of %d bytes...", size)
		data := make([]byte, size)
		io.ReadFull(reader, data)

		decoder := gob.NewDecoder(bytes.NewReader(data))
		var snapshot map[string][]byte
		if err := decoder.Decode(&snapshot); err != nil {
			return err
		}
		s.Import(snapshot)
		log.Println("[replica] full sync completed")

		endLine, _ := reader.ReadString('\n')
		log.Printf("[replica] end marker: %q", strings.TrimSpace(endLine))
	}

	// Step 2: Listen for live updates
	parser := protocol.NewRESPParser()
	for {
		log.Println("[replica] waiting for broadcasted command...")
		parts, err := parser.ParseRESP(reader)
		if err != nil {
			log.Printf("[replica] sync error: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		log.Printf("[replica] received broadcast command: %v", parts)
		applyCommand(s, parts)
	}

}

func applyCommand(s store.IStore, parts []string) {
	cmd := strings.ToUpper(parts[0])
	switch cmd {
	case "SET":
		s.Set(parts[1], []byte(parts[2]), 0)
	case "DEL":
		s.Delete(parts[1])
	default:
		log.Printf("[replica] unknown command %s", cmd)
	}
}
