package util

import (
	"net"
	"strconv"
	"time"
)

func WriteString(conn net.Conn, s string) {
	conn.Write([]byte("+" + s + "\r\n"))
}

func WriteError(conn net.Conn, s string) {
	conn.Write([]byte("-ERR " + s + "\r\n"))
}

func WriteInteger(conn net.Conn, n int) {
	conn.Write([]byte(":" + strconv.Itoa(n) + "\r\n"))
}

const FileVersion = "FDB1"
const NumShards = 16
const FileName = "snapshot.fdb"
const AppendFile = "appendonly.aof"

var StartTime = time.Now()
