package main

import (
	"errors"
	"hash/fnv"
	"log"
	"os"
	"strconv"
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
	clients map[int]*RedisClient
	aeLoop  *AeEventLoop
}

type RedisClient struct {
	fd       int
	db       *RedisDB
	query    string
	args     []*RedisObj // get args from query string
	reply    *List
	sentLen  int
	queryBuf []byte // unhandled query content
	queryLen int    // unhandled query content len
	cmdType  CmdType
	bulkNum  int // number of bulk strings
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
	// resetClient for testcase
	resetClient(c)
}

func freeClient(c *RedisClient) {
	// TODO: delete file event
	// TODO: decrRef reply and args list
	// TODO: delete from clients
}

func resetClient(c *RedisClient) {
	c.cmdType = REDIS_CMD_UNKNOWN

}

func (client *RedisClient) findLineInQuery() (int, error) {
	index := strings.Index(string(client.queryBuf[:client.queryLen]), "\r\n")
	if index < 0 && client.queryLen > REDIS_INLINE_MAX {
		return index, errors.New("too big inline cmd")
	}
	return index, nil
}

// getBulkNumInQuery get number in bulk string, "*3\r\n..." or "$3\r\n..." will return 3.
func (client *RedisClient) getBulkNumInQuery(start, end int) (int, error) {
	num, err := strconv.Atoi(string(client.queryBuf[start:end]))
	client.queryBuf = client.queryBuf[end+2:]
	client.queryLen -= end + 2
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
	// TODO: read query from client
	// TODO: trans query -> args
	// TODO: process command
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
	err = processQueryBuf(c)
	if err != nil {
		log.Printf("handle query buf err: %v\n", err)
		freeClient(c)
		return
	}
}

func SendReplyToClient(el *AeEventLoop, fd int, client interface{}) {
	c := client.(*RedisClient)
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

func RedisStrHash(key *RedisObj) int {
	if key.Type_ != REDISSTR {
		return 0
	}
	hash := fnv.New32()
	hash.Write([]byte(key.StrVal()))
	return int(hash.Sum32())
}

func CreateClient(fd int) *RedisClient {
	var c RedisClient
	c.fd = fd
	c.db = server.db
	c.queryBuf = make([]byte, REDIS_IOBUF_LEN, REDIS_IOBUF_LEN)
	c.reply = ListCreate(ListType{EqualFunc: RedisStrEqual})
	return &c
}

func initServer(config *Config) error {
	server.port = config.Port
	server.addr = config.Addr
	server.clients = make(map[int]*RedisClient)
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
	server.clients[cfd] = c
	server.aeLoop.AeCreateFileEvent(cfd, AE_READABLE, ReadQueryFromClient, nil)
}

const EXPIRE_CHECK_COUNT int = 100

// ServerCron delete key randomly
func ServerCron(loop *AeEventLoop, id int, extra interface{}) {
	for i := 0; i < EXPIRE_CHECK_COUNT; i++ {
		key, val := server.db.expire.RandomGet()
		if key == nil {
			break
		}
		if int64(val.IntVal()) < time.Now().Unix() {
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
