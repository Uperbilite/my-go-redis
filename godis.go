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
	aeLoop  *AeEventLoop
}

type RedisClient struct {
	fd    int
	db    *RedisDB
	query string
	args  []*RedisObj
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

func ReadQueryFromClient(el *AeEventLoop, fd int, client interface{}) {
	// TODO
}

func EqualRedisClient(a, b interface{}) bool {
	c1, ok := a.(*RedisClient)
	if !ok {
		return false
	}
	c2, ok := b.(*RedisClient)
	if !ok {
		return false
	}
	return c1.fd == c2.fd
}

func EqualRedisStr(a, b interface{}) bool {
	o1, ok := a.(*RedisObj)
	if !ok || o1.Type_ != REDISSTR {
		return false
	}
	o2, ok := b.(*RedisObj)
	if !ok || o2.Type_ != REDISSTR {
		return false
	}
	return o1.Val_.(string) == o2.Val_.(string)
}

func RedisStrHash(key interface{}) int {
	o, ok := key.(*RedisObj)
	if !ok || o.Type_ != REDISSTR {
		return 0
	}
	// TODO: compute hash val
	return 0
}

func CreateClient(fd int) *RedisClient {
	var c RedisClient
	c.fd = fd
	c.db = server.db
	c.reply = ListCreate(ListType{EqualFunc: EqualRedisStr})
	server.aeLoop.AeCreateFileEvent(fd, AE_READABLE, ReadQueryFromClient, nil)
	return &c
}

func initServer(config *Config) error {
	server.port = config.Port
	server.addr = config.Addr
	server.clients = ListCreate(ListType{EqualFunc: EqualRedisClient})
	server.db = &RedisDB{
		data: DictCreate(DictType{
			HashFunction: RedisStrHash,
			KeyCompare:   EqualRedisStr,
		}),
		expire: DictCreate(DictType{
			HashFunction: RedisStrHash,
			KeyCompare:   EqualRedisStr,
		}),
	}
	var err error
	server.fd, err = TcpServer(server.port, server.addr)
	server.aeLoop = AeCreateEventLoop()
	return err
}

func main() {
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
