<p align="center">
  <img src="https://github.com/PetarGeorgiev-hash/flashdb/raw/main/assets/flashdb-logo.png" width="160" alt="FlashDB Logo">
</p>

<h1 align="center">FlashDB</h1>

<p align="center">
  <b>High-performance, Redis-compatible in-memory database written in Go.</b>
</p>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/PetarGeorgiev-hash/flashdb">
    <img src="https://goreportcard.com/badge/github.com/PetarGeorgiev-hash/flashdb" alt="Go Report Card">
  </a>
  <a href="https://github.com/PetarGeorgiev-hash/flashdb/actions">
    <img src="https://img.shields.io/github/actions/workflow/status/PetarGeorgiev-hash/flashdb/go.yml?label=build&logo=github" alt="Build Status">
  </a>
  <a href="https://github.com/PetarGeorgiev-hash/flashdb/releases">
    <img src="https://img.shields.io/github/v/release/PetarGeorgiev-hash/flashdb?color=blue&label=release" alt="Release">
  </a>
  <a href="https://github.com/PetarGeorgiev-hash/flashdb/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/PetarGeorgiev-hash/flashdb" alt="License">
  </a>
</p>

---

## Overview

**FlashDB** is a lightweight, high-speed key-value database built in Go.  
It provides a Redis-compatible wire protocol (RESP2), which makes it work seamlessly with tools like `redis-cli`.  
FlashDB is designed for developers who want a minimal, performant, and transparent in-memory data store with persistence.

---

## Key Features

- ⚡ **In-memory speed** with optional persistence
- 💾 **Append-only file (AOF)** durability
- 🧊 **Snapshot-based recovery** for resilience
- ⏱️ **Key expiration (TTL)** support
- 🔌 **RESP2 protocol** — compatible with Redis clients
- 🧩 **Thread-safe sharded store** for concurrency
- 🧠 **Modular command structure** for easy extension

---

## Getting Started

### Prerequisites

- Go 1.21 or later
- (Optional) Docker if you want to run it in a containerized environment

### Build and Run

```bash
git clone https://github.com/PetarGeorgiev-hash/flashdb.git
cd flashdb
go build -o flashdb
./flashdb
```

### Using redis-cli

```bash
redis-cli -p 6379
```

Example session:

```bash
127.0.0.1:6379> SET foo bar
OK
127.0.0.1:6379> GET foo
"bar"
127.0.0.1:6379> DEL foo
1
127.0.0.1:6379> SAVE
OK
```

### Commands

| Command               | Description                                |
| --------------------- | ------------------------------------------ |
| `SET key value [ttl]` | Set a key with an optional expiration time |
| `GET key`             | Retrieve the value of a key                |
| `DEL key`             | Delete a key                               |
| `EXISTS key`          | Check if a key exists                      |
| `TTL key`             | Show remaining time-to-live for a key      |
| `EXPIRE key seconds`  | Set expiration time for a key              |
| `SAVE`                | Create a snapshot and reset the AOF log    |

### Running Test

```bash
go test -v -race ./...
```

### Contributing

We welcome contributions!
If you’d like to add new commands, improve performance, or enhance documentation:

1.Fork the repository
2.Create a new feature branch
3.Submit a pull request
!Please ensure all tests pass before submitting!
