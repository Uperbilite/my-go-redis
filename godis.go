package main

import (
	"log"
	"os"
)

type RedisDB struct {
	data   *Dict
	expire *Dict
}

type RedisServer struct {
	fd      int
	port    int
	addr    string
	db      *RedisDB
	clients *List
	aeLoop  *AeLoop
}

type RedisClient struct {
	fd    int
	db    *RedisDB
	query string
	reply *List
}

type CommandProc func(c *RedisClient)

type RedisCommand struct {
	name  string
	proc  CommandProc
	arity int
}

var server RedisServer
var cmdTable []RedisCommand

func initServer(config *Config) error {
	server.port = config.Port
	server.addr = config.Addr
	server.clients = ListCreate()
	server.db = &RedisDB{
		data:   DictCreate(),
		expire: DictCreate(),
	}
	var err error
	server.fd, err = TcpServer(server.port, server.addr)
	return err
}

func getCommand(c *RedisClient) {
	// TODO
}

func setCommand(c *RedisClient) {
	// TODO
}

func initCmdTable() {
	cmdTable = []RedisCommand{
		{"get", getCommand, 2},
		{"set", setCommand, 3},
	}
}

func main() {
	// TODO: load config and init server
	path := os.Args[1]
	config, err := LoadConfig(path)
	if err != nil {
		log.Printf("Config error: %v\n", err)
	}
	err = initServer(config)
	if err != nil {
		log.Printf("Init server error: %v\n", err)
	}
	initCmdTable()
	log.Println("Redis server is up.")
	server.aeLoop.AeMain()
}
