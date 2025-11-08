package server

import (
	"bufio"
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/PetarGeorgiev-hash/flashdb/aof"
	"github.com/PetarGeorgiev-hash/flashdb/cmd"
	"github.com/PetarGeorgiev-hash/flashdb/protocol"
	"github.com/PetarGeorgiev-hash/flashdb/store"
	"github.com/PetarGeorgiev-hash/flashdb/util"
)

func Start() {

	addr := os.Getenv("FLASHDB_ADDR")
	if addr == "" {
		addr = ":6379"
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

	store := store.NewStore()
	// defer store.Close()

	aofWriter, err := aof.NewAOF(util.AppendFile)
	if err != nil {
		log.Println(err)
	}
	// defer aofWriter.Close()

	go autoSave(store, aofWriter)

	log.Println("Server is listening on port " + addr)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	go func() {
		<-ctx.Done()
		log.Println("[server] shutdown signal received")
		listener.Close()
		store.Close()
		aofWriter.Close()
	}()
	for {
		connection, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				log.Println("[server]  listener closed, exiting")
				return
			default:
				log.Println(err)
				continue
			}
		}
		go handleConnection(connection, store, aofWriter)
	}

}

func handleConnection(conn net.Conn, store store.IStore, aofWriter aof.IAOF) {
	defer conn.Close()

	parser := protocol.NewRESPParser()
	reader := bufio.NewReader(conn)
	for {
		parts, err := parser.ParseRESP(reader)
		log.Printf("[DEBUG] Parsed command: %#v\n", parts)
		if err != nil {
			log.Println("-Error reading from connection:", err.Error())
			return

		}
		if len(parts) == 0 {
			continue
		}

		command := strings.ToUpper(parts[0])

		if handler, ok := cmd.CommandHandlers[command]; ok {
			handler(conn, store, parts, aofWriter)
		} else {
			conn.Write([]byte("-ERR unknown command\r\n"))
		}
	}

}

func autoSave(s store.IStore, aof aof.IAOF) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.StopChan():
			return
		case <-ticker.C:
			if err := s.Save(util.FileName); err != nil {
				log.Printf("[autosave] snapshot save failed: %v", err)
			}
			if err := aof.Reset(); err != nil {
				log.Printf("[autosave] AOF reset failed: %v", err)
			}
		}
	}
}
