<p align="center">
  <img src="https://github.com/PetarGeorgiev-hash/flashdb/raw/main/assets/flashdb-logo.png" width="180" alt="FlashDB Logo">
</p>

<h1 align="center">⚡ FlashDB</h1>

<p align="center">
  <b>A blazing-fast, Redis-compatible in-memory key-value store built in Go.</b>
</p>

<p align="center">
  <a href="https://goreportcard.com/badge/github.com/PetarGeorgiev-hash/flashdb" alt="Go Report Card"></a>
  <img src="https://img.shields.io/github/actions/workflow/status/PetarGeorgiev-hash/flashdb/go.yml?label=build" alt="Build">
  <img src="https://img.shields.io/github/license/PetarGeorgiev-hash/flashdb" alt="License">
  <img src="https://img.shields.io/badge/made%20with-Go-00ADD8.svg" alt="Made with Go">
</p>

---

## 🚀 Features

- ⚡ **In-memory speed** — designed for performance and low latency
- 💾 **AOF persistence** — append-only log for durability
- 🧊 **Snapshot saving** — periodic background saves to disk
- 🕒 **TTL support** — automatic key expiration
- 🔌 **Redis protocol compatible (RESP2)** — works with `redis-cli`
- 🧩 **Modular commands** — easy to extend with new features
- 🧠 **Thread-safe sharded store** for high concurrency

---

## 🧰 Installation

```bash
git clone https://github.com/PetarGeorgiev-hash/flashdb.git
cd flashdb
go build -o flashdb
```
