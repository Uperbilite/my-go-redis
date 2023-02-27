package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func ReadQuery(c *RedisClient, query string) {
	for _, v := range []byte(query) {
		c.queryBuf[c.queryLen] = v
		c.queryLen += 1
	}
}

func TestInlineCmdBuf(t *testing.T) {
	c := CreateClient(0)
	ReadQuery(c, "set key val\r\n")
	ok, err := handleInlineCmdBuf(c)
	assert.Nil(t, err)
	assert.Equal(t, true, ok)
	assert.Equal(t, 3, len(c.args))

	ReadQuery(c, "set ")
	ok, err = handleInlineCmdBuf(c)
	assert.Nil(t, err)
	assert.Equal(t, false, ok)

	ReadQuery(c, "key ")
	ok, err = handleInlineCmdBuf(c)
	assert.Nil(t, err)
	assert.Equal(t, false, ok)

	ReadQuery(c, "val\r\n")
	ok, err = handleInlineCmdBuf(c)
	assert.Nil(t, err)
	assert.Equal(t, true, ok)

	assert.Equal(t, 3, len(c.args))

}

func TestBulkCmdBuf(t *testing.T) {
	c := CreateClient(0)

	ReadQuery(c, "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$3\r\nval\r\n")
	ok, err := handleBulkCmdBuf(c)
	assert.Nil(t, err)
	assert.Equal(t, true, ok)
	assert.Equal(t, 3, len(c.args))

	ReadQuery(c, "*3\r")
	ok, err = handleBulkCmdBuf(c)
	assert.Nil(t, err)
	assert.Equal(t, false, ok)

	ReadQuery(c, "\n$3\r\nSET\r\n$3")
	ok, err = handleBulkCmdBuf(c)
	assert.Nil(t, err)
	assert.Equal(t, false, ok)

	ReadQuery(c, "\r\nkey\r")
	ok, err = handleBulkCmdBuf(c)
	assert.Nil(t, err)
	assert.Equal(t, false, ok)

	ReadQuery(c, "\n$3\r\nval\r\n")
	ok, err = handleBulkCmdBuf(c)
	assert.Nil(t, err)
	assert.Equal(t, true, ok)

	assert.Equal(t, 3, len(c.args))
}

func TestProcessQueryBuf(t *testing.T) {
	conf, _ := LoadConfig("config.json")
	initServer(conf)
	c := CreateClient(server.fd)
	ReadQuery(c, "set key val\r\n")
	err := processQueryBuf(c)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(c.args))

	key := CreateObject(REDISSTR, "key")
	val := server.db.data.DictGet(key)
	assert.Equal(t, "val", val.StrVal())

	ReadQuery(c, "set key val2\r\n")
	err = processQueryBuf(c)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(c.args))
	val2 := server.db.data.DictGet(key)
	assert.Equal(t, "val2", val2.StrVal())

	ReadQuery(c, "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$3\r\nval\r\n")
	err = processQueryBuf(c)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(c.args))
}
