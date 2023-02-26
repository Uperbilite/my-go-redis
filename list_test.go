package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestList(t *testing.T) {
	l := ListCreate(ListType{EqualFunc: RedisStrEqual})
	assert.Equal(t, l.ListLength(), 0)

	l.ListAddNodeTail(CreateObject(REDISSTR, "1"))
	l.ListAddNodeTail(CreateObject(REDISSTR, "2"))
	l.ListAddNodeTail(CreateObject(REDISSTR, "3"))
	assert.Equal(t, l.ListLength(), 3)
	assert.Equal(t, l.ListFirst().Val.Val_.(string), "1")
	assert.Equal(t, l.ListLast().Val.Val_.(string), "3")

	o1 := CreateObject(REDISSTR, "0")
	l.ListAddNodeHead(o1)
	assert.Equal(t, l.ListLength(), 4)
	assert.Equal(t, l.ListFirst().Val.Val_.(string), "0")

	o2 := CreateObject(REDISSTR, "4")
	l.ListAddNodeTail(o2)
	assert.Equal(t, l.ListLength(), 5)
	assert.Equal(t, l.ListLast().Val.Val_.(string), "4")

	n1 := l.ListSearchKey(o1)
	assert.Equal(t, n1.Val, o1)
	n2 := l.ListSearchKey(o2)
	assert.Equal(t, n2.Val, o2)

	l.ListDelKey(o1)
	assert.Equal(t, l.ListLength(), 4)
	n3 := l.ListSearchKey(o1)
	assert.Nil(t, n3)

	l.ListDelNode(l.ListFirst())
	assert.Equal(t, l.ListLength(), 3)
	assert.Equal(t, l.ListFirst().Val.Val_.(string), "2")

	l.ListDelNode(l.ListLast())
	assert.Equal(t, l.ListLength(), 2)
	assert.Equal(t, l.ListLast().Val.Val_.(string), "3")
}
