package main

import (
	"log"
)

type RedisDB struct {
	data   *Dict
	expire *Dict
}

type RedisServer struct {
	fd      int
	port    int
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

func initServer() error {
	// TODO
	return nil
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
	initCmdTable()
	log.Println("Redis server is up.")
	server.aeLoop.AeMain()
}
