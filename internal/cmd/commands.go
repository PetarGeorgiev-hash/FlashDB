package cmd

import (
	"net"
	"strconv"
	"time"

	internal "github.com/PetarGeorgiev-hash/flashdb/internal/store"
)

type CommandHandler func(conn net.Conn, store internal.IStore, parts []string)

var CommandHandlers = map[string]CommandHandler{
	"SET":  handleSet,
	"GET":  handleGet,
	"DEL":  handleDel,
	"PING": handlePing,
}

func handleSet(conn net.Conn, store internal.IStore, parts []string) {
	if len(parts) < 3 {
		conn.Write([]byte("ERR wrong number of arguments for 'SET' command\r\n"))
		return
	}
	key := parts[1]
	value := []byte(parts[2])
	if len(parts) == 4 {
		seconds, err := strconv.Atoi(parts[3])
		if err != nil {
			conn.Write([]byte("Eror invalid expire time\r\n"))
			return
		}
		_, err = store.Set(key, value, time.Duration(seconds)*time.Second)
		if err != nil {
			conn.Write([]byte("Eror failed to set value\r\n"))
			return
		}
		conn.Write([]byte("OK\r\n"))

	} else {
		_, err := store.Set(key, value, 0)
		if err != nil {
			conn.Write([]byte("Eror failed to set value\r\n"))
			return
		}
		conn.Write([]byte("OK\r\n"))
	}
}

func handleGet(conn net.Conn, store internal.IStore, parts []string) {
	if len(parts) < 2 {
		conn.Write([]byte("ERR wrong number of arguments for 'GET' command\r\n"))
		return
	}
	key := parts[1]
	item, err := store.Get(key)
	if err != nil {
		conn.Write([]byte("Eror failed to set value\r\n"))
		return
	}
	if item == nil {
		conn.Write([]byte("(nil)\r\n"))
		return
	}
	conn.Write([]byte(string(item.Value) + "\r\n"))
}

func handleDel(conn net.Conn, store internal.IStore, parts []string) {
	if len(parts) < 2 {
		conn.Write([]byte("ERR wrong number of arguments for 'DEL' command\r\n"))
		return
	}
	key := parts[1]
	err := store.Delete(key)
	if err != nil {
		conn.Write([]byte("Eror failed to delete key or key mismatch\r\n"))
		return
	}
	conn.Write([]byte("(1)\r\n"))
}

func handlePing(conn net.Conn, store internal.IStore, parts []string) {
	if len(parts) == 1 {
		conn.Write([]byte("PONG\r\n"))
	} else {
		conn.Write([]byte(parts[1] + "\r\n"))
	}
}
