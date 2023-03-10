package main

import (
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"strconv"
	"strings"
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
	clients map[int]*RedisClient
	aeLoop  *AeEventLoop
}

type RedisClient struct {
	fd       int
	db       *RedisDB
	args     []*RedisObj
	reply    *List
	sentLen  int
	queryBuf []byte // unhandled query content
	queryLen int    // unhandled query content len
	cmdType  CmdType
	bulkNum  int // number of string in multi bulk command
	bulkLen  int // len of each bulk string
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
	arity int // number of parameter
}

var server RedisServer
var cmdTable []RedisCommand = []RedisCommand{
	{"get", getCommand, 2},
	{"set", setCommand, 3},
	{"expire", expireCommand, 3},
	// TODO: more command
}

func expireIfNeeded(key *RedisObj) {
	entry := server.db.expire.DictFind(key)
	if entry == nil {
		// no expire for this key.
		return
	}
	when := entry.Val.IntVal()
	if when > GetMsTime() {
		// return if the key has not expired.
		return
	}
	server.db.expire.DictDelete(key)
	server.db.data.DictDelete(key)
}

func lookupKeyRead(key *RedisObj) *RedisObj {
	expireIfNeeded(key)
	return server.db.data.DictGet(key)
}

func getCommand(c *RedisClient) {
	key := c.args[1]
	val := lookupKeyRead(key)
	if val == nil {
		// TODO: extract shared.strings
		c.AddReplyStr("$-1\r\n")
	} else if val.Type_ != REDISSTR {
		// TODO: extract shared.strings
		c.AddReplyStr("-ERR: wrong type\r\n")
	} else {
		str := val.StrVal()
		c.AddReplyStr(fmt.Sprintf("$%d%v\t\n", len(str), str))
	}
}

func setCommand(c *RedisClient) {
	key := c.args[1]
	val := c.args[2]
	if val.Type_ != REDISSTR {
		c.AddReplyStr("-ERR: wrong type\r\n")
	}
	server.db.data.DictSet(key, val)
	server.db.expire.DictDelete(key)
	c.AddReplyStr("+OK\r\n")
}

func expireCommand(c *RedisClient) {
	key := c.args[1]
	val := c.args[2]
	if val.Type_ != REDISSTR {
		// TODO: extract shared.strings
		c.AddReplyStr("-ERR: wrong type\r\n")
	}
	expire := GetMsTime() + (val.IntVal() * 1000)
	expObj := CreateFromInt(expire)
	server.db.expire.DictSet(key, expObj)
	expObj.DecrRefCount()
	c.AddReplyStr("+OK\r\n")
}

func lookupCommand(cmdName string) *RedisCommand {
	for _, c := range cmdTable {
		if c.name == cmdName {
			return &c
		}
	}
	return nil
}

func (c *RedisClient) AddReply(obj *RedisObj) {
	c.reply.ListAddNodeTail(obj)
	obj.IncrRefCount()
	server.aeLoop.AeCreateFileEvent(c.fd, AE_WRITABLE, SendReplyToClient, c)
}

func (c *RedisClient) AddReplyStr(str string) {
	obj := CreateObject(REDISSTR, str)
	c.AddReply(obj)
	obj.DecrRefCount()
}

func processCommand(c *RedisClient) {
	cmdName := c.args[0].StrVal()
	log.Printf("process command: %v\n", cmdName)
	if cmdName == "quit" {
		freeClient(c)
		return
	}
	cmd := lookupCommand(cmdName)
	if cmd == nil {
		c.AddReplyStr("-ERR: unknown command\r\n")
		resetClient(c)
		return
	} else if cmd.arity != len(c.args) {
		c.AddReplyStr("-ERR: wrong number of args\r\n")
		resetClient(c)
		return
	}
	cmd.proc(c)
	resetClient(c)
}

func freeClientArgs(c *RedisClient) {
	for _, v := range c.args {
		v.DecrRefCount()
	}
}

func freeReplyList(c *RedisClient) {
	for c.reply.length != 0 {
		n := c.reply.head
		c.reply.ListDelNode(n)
		n.Val.DecrRefCount()
	}
}

func freeClient(c *RedisClient) {
	freeClientArgs(c)
	freeReplyList(c)
	delete(server.clients, c.fd)
	server.aeLoop.AeDeleteFileEvent(c.fd, AE_READABLE)
	server.aeLoop.AeDeleteFileEvent(c.fd, AE_WRITABLE)
	Close(c.fd)
}

func resetClient(c *RedisClient) {
	freeClientArgs(c)
	c.cmdType = REDIS_CMD_UNKNOWN
	c.bulkNum = 0
	c.bulkLen = 0
}

func (c *RedisClient) findLineInQuery() (int, error) {
	index := strings.Index(string(c.queryBuf[:c.queryLen]), "\r\n")
	if index < 0 && c.queryLen > REDIS_INLINE_MAX {
		return index, errors.New("too big inline cmd")
	}
	return index, nil
}

// getBulkNumInQuery get number in bulk string, "*3\r\n..." or "$3\r\n..." will return 3.
func (c *RedisClient) getBulkNumInQuery(start, end int) (int, error) {
	num, err := strconv.Atoi(string(c.queryBuf[start:end]))
	c.queryBuf = c.queryBuf[end+2:]
	c.queryLen -= end + 2
	return num, err
}

func handleInlineCmdBuf(c *RedisClient) (bool, error) {
	index, err := c.findLineInQuery()
	if index < 0 {
		return false, err
	}

	subs := strings.Split(string(c.queryBuf[:index]), " ")
	c.queryBuf = c.queryBuf[index+2:] // plus 2 to skip "/r/n"
	c.queryLen -= index + 2
	c.args = make([]*RedisObj, len(subs), len(subs))
	for i, v := range subs {
		c.args[i] = CreateObject(REDISSTR, v)
	}

	return true, nil
}

func handleBulkCmdBuf(c *RedisClient) (bool, error) {
	// read bulk num
	if c.bulkNum == 0 {
		index, err := c.findLineInQuery()
		if index < 0 {
			return false, err
		}
		bnum, err := c.getBulkNumInQuery(1, index)
		if err != nil {
			return false, err
		}
		if bnum == 0 {
			return true, nil
		}
		c.bulkNum = bnum
		c.args = make([]*RedisObj, bnum, bnum)
	}

	// read every bulk string
	for c.bulkNum > 0 {
		// read bulk length
		if c.bulkLen == 0 {
			index, err := c.findLineInQuery()
			if index < 0 {
				return false, err
			}
			if c.queryBuf[0] != '$' {
				return false, errors.New("expect $ for bulk length")
			}
			blen, err := c.getBulkNumInQuery(1, index)
			if err != nil || blen == 0 {
				return false, err
			}
			if blen > REDIS_BULK_MAX {
				return false, errors.New("too big bulk")
			}
			c.bulkLen = blen
		}
		// read bulk string
		if c.queryLen < c.bulkLen+2 {
			return false, nil
		}
		index := c.bulkLen
		if c.queryBuf[index] != '\r' || c.queryBuf[index+1] != '\n' {
			return false, errors.New("expect CRLF for bulk string end")
		}
		c.args[len(c.args)-c.bulkNum] = CreateObject(REDISSTR, string(c.queryBuf[:index]))
		c.queryBuf = c.queryBuf[index+2:]
		c.queryLen -= index + 2
		c.bulkLen = 0
		c.bulkNum -= 1
	}
	// read every bulk
	return true, nil
}

func processQueryBuf(c *RedisClient) error {
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
				// accept empty command
				resetClient(c)
			} else {
				processCommand(c)
			}
		} else {
			// command incomplete
			break
		}
	}
	return nil
}

func ReadQueryFromClient(el *AeEventLoop, fd int, client interface{}) {
	c := client.(*RedisClient)
	if len(c.queryBuf)-c.queryLen < REDIS_BULK_MAX {
		c.queryBuf = append(c.queryBuf, make([]byte, REDIS_BULK_MAX, REDIS_BULK_MAX)...)
	}
	n, err := Read(fd, c.queryBuf[c.queryLen:])
	if err != nil {
		log.Printf("client %v read err: %v\n", fd, err)
		freeClient(c)
		return
	}
	c.queryLen += n
	log.Printf("read %v bytes from client: %v\n", n, c.fd)
	log.Printf("ReadQueryFromClient, queryBuf: %v\n", string(c.queryBuf))
	err = processQueryBuf(c)
	if err != nil {
		log.Printf("handle query buf err: %v\n", err)
		freeClient(c)
		return
	}
}

func SendReplyToClient(el *AeEventLoop, fd int, client interface{}) {
	c := client.(*RedisClient)
	log.Printf("SendReplyToClient, reply len: %v\n", c.reply.ListLength())
	for c.reply.ListLength() > 0 {
		rep := c.reply.ListFirst()
		buf := []byte(rep.Val.StrVal())
		bufLen := len(buf)
		if c.sentLen < bufLen {
			n, err := Write(fd, buf[c.sentLen:])
			if err != nil {
				log.Printf("send reply err: %v\n", err)
				freeClient(c)
				return
			}
			c.sentLen += n
			log.Printf("send %v bytes to client: %v\n", n, c.fd)
			if c.sentLen == bufLen {
				c.reply.ListDelNode(rep)
				rep.Val.DecrRefCount()
				c.sentLen = 0
			} else {
				break
			}
		}
	}
	if c.reply.ListLength() == 0 {
		c.sentLen = 0
		el.AeDeleteFileEvent(fd, AE_WRITABLE)
	}
}

func RedisStrEqual(a, b *RedisObj) bool {
	if a.Type_ != REDISSTR || b.Type_ != REDISSTR {
		return false
	}
	return a.StrVal() == b.StrVal()
}

func RedisStrHash(key *RedisObj) int64 {
	if key.Type_ != REDISSTR {
		return 0
	}
	hash := fnv.New64()
	hash.Write([]byte(key.StrVal()))
	return int64(hash.Sum64())
}

func CreateClient(fd int) *RedisClient {
	var c RedisClient
	c.fd = fd
	c.db = server.db
	c.queryBuf = make([]byte, REDIS_IOBUF_LEN, REDIS_IOBUF_LEN)
	c.reply = ListCreate(ListFunc{EqualFunc: RedisStrEqual})
	return &c
}

func initServer(config *Config) error {
	server.port = config.Port
	server.addr = config.Addr
	server.clients = make(map[int]*RedisClient)
	server.db = &RedisDB{
		data: DictCreate(DictFunc{
			HashFunc:  RedisStrHash,
			EqualFunc: RedisStrEqual,
		}),
		expire: DictCreate(DictFunc{
			HashFunc:  RedisStrHash,
			EqualFunc: RedisStrEqual,
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
	server.clients[cfd] = c
	server.aeLoop.AeCreateFileEvent(cfd, AE_READABLE, ReadQueryFromClient, c)
	log.Printf("accept client, fd: %v\n", cfd)
}

const EXPIRE_CHECK_COUNT int = 100

// ServerCron delete key randomly
func ServerCron(loop *AeEventLoop, id int, extra interface{}) {
	for i := 0; i < EXPIRE_CHECK_COUNT; i++ {
		entry := server.db.expire.DictGetRandomKey()
		if entry == nil {
			break
		}
		if entry.Val.IntVal() < GetMsTime() {
			server.db.data.DictDelete(entry.Key)
			server.db.expire.DictDelete(entry.Key)
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
