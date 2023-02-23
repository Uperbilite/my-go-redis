package main

import (
	"hash/fnv"
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

func RedisClientEqual(a, b interface{}) bool {
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

func RedisStrEqual(a, b interface{}) bool {
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
	hash := fnv.New32()
	hash.Write([]byte(o.Val_.(string)))
	return int(hash.Sum32())
}

func CreateClient(fd int) *RedisClient {
	var c RedisClient
	c.fd = fd
	c.db = server.db
	c.reply = ListCreate(ListType{EqualFunc: RedisStrEqual})
	server.aeLoop.AeCreateFileEvent(fd, AE_READABLE, ReadQueryFromClient, nil)
	return &c
}

func initServer(config *Config) error {
	server.port = config.Port
	server.addr = config.Addr
	server.clients = ListCreate(ListType{EqualFunc: RedisClientEqual})
	server.db = &RedisDB{
		data: DictCreate(DictType{
			HashFunction: RedisStrHash,
			KeyCompare:   RedisStrEqual,
		}),
		expire: DictCreate(DictType{
			HashFunction: RedisStrHash,
			KeyCompare:   RedisStrEqual,
		}),
	}
	var err error
	server.fd, err = TcpServer(server.port, server.addr)
	server.aeLoop = AeCreateEventLoop()
	return err
}

func AcceptHandler(le *AeEventLoop, fd int, extra interface{}) {
	cfd, err := Accept(fd)
	if err != nil {
		log.Printf("accept err: %v\n", err)
		return
	}
	c := CreateClient(cfd)
	// TODO: check max clients limit
	server.clients.ListAddNodeHead(c)
}

// background cron per 1000ms
func ServerCron(loop *AeEventLoop, id int, extra interface{}) {
	// TODO: background job
}

func main() {
	path := os.Args[1]
	config, err := LoadConfig(path)
	if err != nil {
		log.Printf("Config error: %v\n", err)
	}
	initCmdTable()
	err = initServer(config)
	if err != nil {
		log.Printf("Init server error: %v\n", err)
	}
	server.aeLoop.AeCreateFileEvent(server.fd, AE_READABLE, AcceptHandler, nil)
	server.aeLoop.AeCreateTimeEvent(AE_NORMAL, 1000, ServerCron, nil)
	log.Println("Redis server is up.")
	server.aeLoop.AeMain()
}
