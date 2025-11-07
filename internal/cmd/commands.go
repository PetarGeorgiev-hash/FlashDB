package cmd

import (
	"net"
	"strconv"
	"time"

	internal "github.com/PetarGeorgiev-hash/flashdb/internal/store"
	"github.com/PetarGeorgiev-hash/flashdb/internal/util"
)

const (
	SetCommand    = "SET"
	GetCommand    = "GET"
	DelCommand    = "DEL"
	PingCommand   = "PING"
	ExistsCommand = "EXISTS"
	TTLCommand    = "TTL"
	ExpireCommand = "EXPIRE"
	SaveCommand   = "SAVE"
)

type CommandHandler func(conn net.Conn, store internal.IStore, parts []string)

var CommandHandlers = map[string]CommandHandler{
	SetCommand:    handleSet,
	GetCommand:    handleGet,
	DelCommand:    handleDel,
	PingCommand:   handlePing,
	ExistsCommand: handleExists,
	TTLCommand:    handleTTL,
	ExpireCommand: handleExpire,
	SaveCommand:   handleSave,
}

func handleSet(conn net.Conn, store internal.IStore, parts []string) {
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
		util.WriteString(conn, "OK")
	}
}

func handleGet(conn net.Conn, store internal.IStore, parts []string) {
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

func handleDel(conn net.Conn, store internal.IStore, parts []string) {
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
	conn.Write([]byte(":1\r\n"))
}

func handlePing(conn net.Conn, store internal.IStore, parts []string) {
	if len(parts) == 1 {
		conn.Write([]byte("PONG\r\n"))
	} else {
		conn.Write([]byte(parts[1] + "\r\n"))
	}
}

func handleExists(conn net.Conn, store internal.IStore, parts []string) {
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

func handleTTL(conn net.Conn, store internal.IStore, parts []string) {
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

func handleExpire(conn net.Conn, store internal.IStore, parts []string) {
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
	util.WriteInteger(conn, 1) // Expiration set successfully
}

func handleSave(conn net.Conn, store internal.IStore, parts []string) {
	err := store.Save(util.FileName)
	if err != nil {
		util.WriteError(conn, "failed to save data to disk"+err.Error())
		return
	}
	util.WriteString(conn, "OK")
}
