package main

import (
	"bufio"
	"log"
	"net"
	"strings"

	"github.com/PetarGeorgiev-hash/flashdb/internal/cmd"
	"github.com/PetarGeorgiev-hash/flashdb/internal/protocol"
	internal "github.com/PetarGeorgiev-hash/flashdb/internal/store"
)

func main() {
	listener, err := net.Listen("tcp", ":6380")
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

	store := internal.NewStore()
	log.Println("Server is listening on port 6380")
	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleConnection(connection, store)
	}

}

func handleConnection(conn net.Conn, store internal.IStore) {
	defer conn.Close()

	parser := protocol.NewRESPParser()
	reader := bufio.NewReader(conn)
	for {
		// line, err := reader.ReadString('\n')
		// addr := conn.RemoteAddr().String()
		// log.Printf("[%s] Received command: %s", addr, line)
		// if err != nil {
		// 	log.Println("Error reading from connection:", err)
		// 	return
		// }

		// input := strings.TrimSuffix(line, "\r\n")
		// parts := strings.Fields(input)
		// if len(parts) == 0 {
		// 	continue
		// }
		parts, err := parser.ParseRESP(reader)
		if err != nil {
			log.Println("Error reading from connection:", err)
			return

		}
		if len(parts) == 0 {
			continue
		}

		command := strings.ToUpper(parts[0])

		if handler, ok := cmd.CommandHandlers[command]; ok {
			handler(conn, store, parts)
		} else {
			conn.Write([]byte("ERR unknown command\r\n"))
		}
	}

}
