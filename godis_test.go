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

func TestInlineBuf(t *testing.T) {
	c := CreateClient(0)
	ReadQuery(c, "set key val\r\n")
	ok, err := handleInlineCmdBuf(c)
	assert.Nil(t, err)
	assert.Equal(t, true, ok)

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
	assert.Equal(t, false, ok)

	assert.Equal(t, 3, len(c.args))

}
