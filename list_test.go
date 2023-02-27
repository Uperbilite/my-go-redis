package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestList(t *testing.T) {
	l := ListCreate(ListFunc{EqualFunc: RedisStrEqual})
	assert.Equal(t, 0, l.ListLength())

	l.ListAddNodeTail(CreateObject(REDISSTR, "4"))
	l.ListDelNode(l.ListFirst())

	l.ListAddNodeTail(CreateObject(REDISSTR, "1"))
	l.ListAddNodeTail(CreateObject(REDISSTR, "2"))
	l.ListAddNodeTail(CreateObject(REDISSTR, "3"))
	assert.Equal(t, 3, l.ListLength())
	assert.Equal(t, "1", l.ListFirst().Val.StrVal())
	assert.Equal(t, "3", l.ListLast().Val.StrVal())

	o1 := CreateObject(REDISSTR, "0")
	l.ListAddNodeHead(o1)
	assert.Equal(t, 4, l.ListLength())
	assert.Equal(t, "0", l.ListFirst().Val.StrVal())

	o2 := CreateObject(REDISSTR, "4")
	l.ListAddNodeTail(o2)
	assert.Equal(t, 5, l.ListLength())
	assert.Equal(t, "4", l.ListLast().Val.StrVal())

	n1 := l.ListSearchKey(o1)
	assert.Equal(t, o1, n1.Val)
	n2 := l.ListSearchKey(o2)
	assert.Equal(t, o2, n2.Val)

	l.ListDelKey(o1)
	assert.Equal(t, 4, l.ListLength())
	assert.Nil(t, l.ListSearchKey(o1))

	l.ListDelNode(l.ListFirst())
	assert.Equal(t, 3, l.ListLength())
	assert.Equal(t, "2", l.ListFirst().Val.StrVal())

	l.ListDelNode(l.ListLast())
	assert.Equal(t, 2, l.ListLength())
	assert.Equal(t, "3", l.ListLast().Val.StrVal())

	l.ListDelNode(l.ListFirst())
	assert.Equal(t, 1, l.ListLength())
	assert.Equal(t, "3", l.ListFirst().Val.StrVal())
	assert.Equal(t, l.ListFirst(), l.ListLast())

	l.ListDelNode(l.ListLast())
	assert.Equal(t, 0, l.ListLength())
	assert.Nil(t, l.ListFirst())
	assert.Nil(t, l.ListLast())

	l.ListDelNode(l.ListLast())
	assert.Equal(t, 0, l.ListLength())
}
