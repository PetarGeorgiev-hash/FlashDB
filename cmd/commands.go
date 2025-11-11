package cmd

import (
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/PetarGeorgiev-hash/flashdb/aof"
	"github.com/PetarGeorgiev-hash/flashdb/replication"
	internal "github.com/PetarGeorgiev-hash/flashdb/store"
	"github.com/PetarGeorgiev-hash/flashdb/util"
)

const (
	SetCommand     = "SET"
	GetCommand     = "GET"
	DelCommand     = "DEL"
	PingCommand    = "PING"
	ExistsCommand  = "EXISTS"
	TTLCommand     = "TTL"
	ExpireCommand  = "EXPIRE"
	SaveCommand    = "SAVE"
	InfoCommand    = "INFO"
	CommandCommand = "COMMAND"
)

type CommandHandler func(conn net.Conn, store internal.IStore, parts []string, aofWriter aof.IAOF, replManager replication.IManager)

var CommandHandlers = map[string]CommandHandler{
	SetCommand:     handleSet,
	GetCommand:     handleGet,
	DelCommand:     handleDel,
	PingCommand:    handlePing,
	ExistsCommand:  handleExists,
	TTLCommand:     handleTTL,
	ExpireCommand:  handleExpire,
	SaveCommand:    handleSave,
	InfoCommand:    handleInfo,
	CommandCommand: handleCommand,
}

func handleSet(conn net.Conn, store internal.IStore, parts []string, aofWriter aof.IAOF, replManager replication.IManager) {
	if len(parts) < 3 {
		util.WriteError(conn, "wrong number of arguments for 'SET' command")
		return
	}
	key := parts[1]
	value := []byte(parts[2])
	if len(parts) == 4 {
		seconds, err := strconv.Atoi(parts[3])
		if err != nil {
			util.WriteError(conn, "invalid expire time")
			return
		}
		_, err = store.Set(key, value, time.Duration(seconds)*time.Second)
		if err != nil {
			util.WriteError(conn, "failed to set value")
			return
		}

		util.WriteString(conn, "OK")

	} else {
		_, err := store.Set(key, value, 0)
		if err != nil {
			util.WriteError(conn, "failed to set value")
			return
		}
		log.Println("sending ok")
		util.WriteString(conn, "OK")
	}
	if err := aofWriter.AppendCommand(parts...); err != nil {
		log.Printf("[AOF] failed to append command: %v", err)
	}
	// TODO: check if the command is comming from replica replManager == nil pointer and will crash
	replManager.Broadcast(parts)
}

func handleGet(conn net.Conn, store internal.IStore, parts []string, aofWriter aof.IAOF, replManager replication.IManager) {
	if len(parts) < 2 {
		util.WriteError(conn, "wrong number of arguments for 'GET' command")
		return
	}
	key := parts[1]
	item, err := store.Get(key)
	if err != nil {
		util.WriteError(conn, "failed to get value")
		return
	}
	if item == nil {
		conn.Write([]byte("$-1\r\n"))
		return
	}

	conn.Write([]byte("$" + strconv.Itoa(len(item.Value)) + "\r\n"))
	conn.Write(item.Value)
	conn.Write([]byte("\r\n"))
}

func handleDel(conn net.Conn, store internal.IStore, parts []string, aofWriter aof.IAOF, replManager replication.IManager) {
	if len(parts) < 2 {
		util.WriteError(conn, "wrong number of arguments for 'DEL' command")
		return
	}
	key := parts[1]
	err := store.Delete(key)
	if err != nil {
		util.WriteError(conn, "failed to delete key or key mismatch")
		return
	}
	err = aofWriter.AppendCommand(parts...)
	if err != nil {
		util.WriteError(conn, "failed to save aof")
		return
	}
	util.WriteInteger(conn, 1)
	replManager.Broadcast(parts)
}

func handlePing(conn net.Conn, store internal.IStore, parts []string, aofWriter aof.IAOF, replManager replication.IManager) {
	if len(parts) == 1 {
		util.WriteString(conn, "PONG")
	} else {
		util.WriteString(conn, parts[1])
	}
}

func handleExists(conn net.Conn, store internal.IStore, parts []string, aofWriter aof.IAOF, replManager replication.IManager) {
	if len(parts) < 2 {
		util.WriteError(conn, "wrong number of arguments for 'EXISTS' command")
		return
	}
	key := parts[1]
	item, err := store.Get(key)
	if err != nil {
		util.WriteError(conn, "failed to get value")
		return
	}
	if item == nil {
		util.WriteInteger(conn, 0)
		return
	}
	util.WriteInteger(conn, 1)
}

func handleTTL(conn net.Conn, store internal.IStore, parts []string, aofWriter aof.IAOF, replManager replication.IManager) {
	if len(parts) < 2 {
		util.WriteError(conn, "wrong number of arguments for 'TTL' command")
		return
	}
	key := parts[1]
	item, err := store.Get(key)
	if err != nil {
		util.WriteError(conn, "failed to get value")
		return
	}
	if item == nil {
		util.WriteInteger(conn, -2) // Key does not exist
		return
	}

	if item.ExpiresAt.IsZero() {
		util.WriteInteger(conn, -1) // Key exists but has no expiration
		return
	}
	ttl := int(time.Until(item.ExpiresAt).Seconds())
	if ttl < 0 {
		util.WriteInteger(conn, -2) // Key has expired
		return
	}
	util.WriteInteger(conn, ttl) // Return TTL in seconds

}

func handleExpire(conn net.Conn, store internal.IStore, parts []string, aofWriter aof.IAOF, replManager replication.IManager) {
	if len(parts) < 3 {
		util.WriteError(conn, "wrong number of arguments for 'EXPIRE' command")
		return
	}

	key := parts[1]
	seconds, err := strconv.ParseInt(parts[2], 0, 64)
	if err != nil {
		util.WriteError(conn, "invalid seconds") // Invalid seconds
		return
	}

	item, err := store.Get(key)
	if err != nil {
		util.WriteError(conn, "failed to get value")
		return
	}
	if item == nil {
		util.WriteInteger(conn, 0) // Key does not exist
		return
	}
	item.ExpiresAt = time.Now().Add(time.Duration(seconds) * time.Second)
	err = aofWriter.AppendCommand(parts...)
	if err != nil {
		util.WriteError(conn, "failed to save aof")
		return
	}
	util.WriteInteger(conn, 1) // Expiration set successfully
	replManager.Broadcast(parts)
}

func handleSave(conn net.Conn, store internal.IStore, parts []string, aofWriter aof.IAOF, replManager replication.IManager) {
	err := store.Save(util.FileName)
	if err != nil {
		util.WriteError(conn, "failed to save data to disk"+err.Error())
		return
	}
	err = aofWriter.Reset()
	if err != nil {
		util.WriteError(conn, "failed to reset the aof file"+err.Error())
	}
	util.WriteString(conn, "OK")
}

func handleInfo(conn net.Conn, store internal.IStore, parts []string, aofWriter aof.IAOF, replManager replication.IManager) {
	// Simulate Redis INFO output (just minimal subset)
	uptime := int(time.Since(util.StartTime).Seconds())
	info := "# Server\r\n" +
		"redis_version:0.0.1-flashdb\r\n" +
		"uptime_in_seconds:" + strconv.Itoa(uptime) + "\r\n" +
		"arch_bits:64\r\n" +
		"process_id:" + strconv.Itoa(os.Getpid()) + "\r\n" +
		"go_version:" + runtime.Version() + "\r\n" +
		"# Memory\r\n" +
		"mem_allocator:golang\r\n" +
		"# FlashDB\r\n" +
		"store_backend:in-memory\r\n"

	conn.Write([]byte("$" + strconv.Itoa(len(info)) + "\r\n"))
	conn.Write([]byte(info))
	conn.Write([]byte("\r\n"))
}

func handleCommand(conn net.Conn, store internal.IStore, parts []string, aofWriter aof.IAOF, replManager replication.IManager) {
	conn.Write([]byte("*0\r\n"))
}
