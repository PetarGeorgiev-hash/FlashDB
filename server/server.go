package server

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/PetarGeorgiev-hash/flashdb/aof"
	"github.com/PetarGeorgiev-hash/flashdb/cluster"
	"github.com/PetarGeorgiev-hash/flashdb/cmd"
	"github.com/PetarGeorgiev-hash/flashdb/protocol"
	"github.com/PetarGeorgiev-hash/flashdb/replication"
	"github.com/PetarGeorgiev-hash/flashdb/store"
	"github.com/PetarGeorgiev-hash/flashdb/util"
)

func Start() {

	addr := os.Getenv("FLASHDB_ADDR")
	if addr == "" {
		addr = ":6379"
	}
	if !strings.Contains(addr, "127.0.0.1") && strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

	store := store.NewStore()

	aofWriter, err := aof.NewAOF(util.AppendFile)
	if err != nil {
		log.Println(err)
	}

	cfg, err := cluster.LoadConfig("cluster.json")
	if err != nil {
		log.Fatalf("failed to load cluster config: %v", err)
	}

	clusterManager := cluster.NewManager(cfg, addr)

	var replManager replication.IManager
	role := os.Getenv("FLASHDB_ROLE")
	if role == "replica" {
		masterAddr := os.Getenv("FLASHDB_MASTER_ADDR")
		go replication.StartReplica(masterAddr, store)
	} else {
		replManager = replication.NewManager(store)
		go listenForReplicas(replManager, addr)
	}
	err = aofWriter.LoadAOF(util.AppendFile, store)
	if err != nil {
		log.Println(err)
	}

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
		go handleConnection(connection, store, aofWriter, replManager, clusterManager, addr)
	}

}

func handleConnection(conn net.Conn, store store.IStore, aofWriter aof.IAOF, replManager replication.IManager, clusterManager *cluster.Manager, addr string) {
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

		// get the key and compute it then see does this node own it
		// if not return moved and the owner of the slot
		if len(parts) > 1 {
			key := parts[1]
			slot := clusterManager.GetSlotForKey(key)
			owner := clusterManager.GetOwner(slot)
			if owner != "" && owner != addr {
				if !clusterManager.IsLocal(slot) {
					conn.Write([]byte(fmt.Sprintf("-MOVED %d %s\r\n", slot, owner)))
					return
				}
			}
		}

		command := strings.ToUpper(parts[0])

		if handler, ok := cmd.CommandHandlers[command]; ok {
			handler(conn, store, parts, aofWriter, replManager)
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

func listenForReplicas(m replication.IManager, addr string) {
	replicationPort := 10000 + extractPort(addr)
	listenAddr := fmt.Sprintf(":%d", replicationPort)
	ln, err := net.Listen("tcp", listenAddr)

	if err != nil {
		log.Printf("replication listener failed: %v", err)
		return
	}
	log.Printf("[replication] listening on port %v for replicas...", replicationPort)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("[replication] accept error:", err)
			continue
		}
		go m.HandleReplicationConn(conn)
	}
}

func extractPort(addr string) int {
	parts := strings.Split(addr, ":")
	if len(parts) < 2 {
		return 6379
	}
	port, _ := strconv.Atoi(parts[1])
	return port
}
