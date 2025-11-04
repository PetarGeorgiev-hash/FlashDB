package main

import (
	"bufio"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

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

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		log.Println(line)
		if err != nil {
			log.Println("Error reading from connection:", err)
			return
		}
		log.Printf("Received command: %s", line)

		input := strings.TrimSuffix(line, "\r\n")
		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}

		cmd := strings.ToUpper(parts[0])
		switch cmd {
		case "SET":
			if len(parts) < 3 {
				conn.Write([]byte("ERR wrong number of arguments for 'SET' command\r\n"))
				continue
			}
			key := parts[1]
			value := []byte(parts[2])
			if len(parts) == 4 {
				seconds, err := strconv.Atoi(parts[3])
				if err != nil {
					conn.Write([]byte("Eror invalid expire time\r\n"))
					continue
				}
				_, err = store.Set(key, value, time.Duration(seconds)*time.Second)
				if err != nil {
					conn.Write([]byte("Eror failed to set value\r\n"))
					continue
				}
				conn.Write([]byte("OK\r\n"))

			} else {
				_, err := store.Set(key, value, 0)
				if err != nil {
					conn.Write([]byte("Eror failed to set value\r\n"))
					continue
				}
				conn.Write([]byte("OK\r\n"))
			}

		}
	}

}
