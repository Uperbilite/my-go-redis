package main

import (
	"errors"
	"golang.org/x/sys/unix"
	"hash/fnv"
	"log"
	"os"
	"strings"
	"time"
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
	fd       int
	db       *RedisDB
	query    string
	args     []*RedisObj // get args from query string
	reply    *List
	queryBuf []byte
	queryLen int
	cmdType  CmdType
	bulkNum  int
	bulkLen  int
}

type CmdType = byte

const (
	REDIS_CMD_UNKNOWN CmdType = 0x00
	REDIS_CMD_INLINE  CmdType = 0x01
	REDIS_CMD_BULK    CmdType = 0x02
)

const (
	REDIS_IOBUF_LEN  int = 1024 * 16
	REDIS_INLINE_MAX int = 1024 * 4
	REDIS_BULK_MAX   int = 1024 * 4
)

type CommandProc func(c *RedisClient)

type RedisCommand struct {
	name  string
	proc  CommandProc
	arity int
}

var server RedisServer
var cmdTable []RedisCommand = []RedisCommand{
	{"get", getCommand, 2},
	{"set", setCommand, 3},
}

func getCommand(c *RedisClient) {
	// TODO
}

func setCommand(c *RedisClient) {
	// TODO
}

func processCommand(c *RedisClient) {
	// TODO: lookup command
	// TODO: call command
	// TODO: decrRef args
}

func freeClient(c *RedisClient) {
	// TODO: delete file event
	// TODO: decrRef reply and args list
	// TODO: delete from clients
}

func resetClient(c *RedisClient) {

}

func handleInlineCmdBuf(c *RedisClient) (bool, error) {
	index := strings.IndexAny(string(c.queryBuf[:c.queryLen]), "\r\n")
	if index < 0 {
		if c.queryLen > REDIS_INLINE_MAX {
			return false, errors.New("too big inline cmd")
		} else {
			// wait to next read
			return false, nil
		}
	}

	subs := strings.Split(string(c.queryBuf[:index]), " ")
	c.queryBuf = c.queryBuf[index+2:]
	c.queryLen -= index + 2
	c.args = make([]*RedisObj, len(subs), len(subs))
	for i, v := range subs {
		c.args[i] = CreateObject(REDISSTR, v)
	}

	return true, nil
}

func handleBulkCmdBuf(c *RedisClient) (bool, error) {
	return false, nil
}

func handleQueryBuf(c *RedisClient) error {
	for c.queryLen > 0 {
		if c.cmdType == REDIS_CMD_UNKNOWN {
			if c.queryBuf[0] == '*' {
				c.cmdType = REDIS_CMD_BULK
			} else {
				c.cmdType = REDIS_CMD_INLINE
			}
		}
		// trans query to args
		var ok bool
		var err error
		if c.cmdType == REDIS_CMD_INLINE {
			ok, err = handleInlineCmdBuf(c)
		} else if c.cmdType == REDIS_CMD_BULK {
			ok, err = handleBulkCmdBuf(c)
		} else {
			return errors.New("unknown command type")
		}
		if err != nil {
			return err
		}
		if ok {
			if len(c.args) == 0 {
				resetClient(c)
			} else {
				processCommand(c)
			}
		} else {
			break
		}
	}
	return nil
}

func ReadQueryFromClient(el *AeEventLoop, fd int, client interface{}) {
	// TODO: read query from client
	// TODO: trans query -> args
	// TODO: process command
	c := client.(*RedisClient)
	if len(c.queryBuf)-c.queryLen < REDIS_BULK_MAX {
		c.queryBuf = append(c.queryBuf, make([]byte, REDIS_BULK_MAX, REDIS_BULK_MAX)...)
	}
	n, err := unix.Read(fd, c.queryBuf[c.queryLen:])
	if err != nil {
		log.Printf("client %v read err: %v\n", fd, err)
		return
	}
	c.queryLen += n
	err = handleQueryBuf(c)
	if err != nil {
		log.Printf("handle query buf err: %v\n", err)
		return
	}
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
	c.queryBuf = make([]byte, REDIS_IOBUF_LEN, REDIS_IOBUF_LEN)
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
	server.aeLoop, err = AeCreateEventLoop()
	if err != nil {
		return err
	}

	server.fd, err = TcpServer(server.port, server.addr)

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

const EXPIRE_CHECK_COUNT int = 100

// ServerCron delete key randomly
func ServerCron(loop *AeEventLoop, id int, extra interface{}) {
	for i := 0; i < EXPIRE_CHECK_COUNT; i++ {
		key, val := server.db.expire.RandomGet()
		if key == nil {
			break
		}
		if int64(val.(*RedisObj).IntVal()) < time.Now().Unix() {
			server.db.data.DeleteKey(key)
			server.db.expire.DeleteKey(key)
		}
	}
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
	server.aeLoop.AeCreateFileEvent(server.fd, AE_READABLE, AcceptHandler, nil)
	server.aeLoop.AeCreateTimeEvent(AE_NORMAL, 1, ServerCron, nil)
	log.Println("Redis server is up.")
	server.aeLoop.AeMain()
}
