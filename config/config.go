package config

type Config struct {
	Addr       string
	Role       string // "master" or "replica"
	MasterAddr string // use when role == "replica"
}
