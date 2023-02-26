package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDict(t *testing.T) {
	d := DictCreate(DictType{
		HashFunction: RedisStrHash,
		KeyCompare:   RedisStrEqual,
	})

	entry := d.DictGetRandomKey()
	assert.Nil(t, entry)

	k1 := CreateObject(REDISSTR, "k1")
	v1 := CreateObject(REDISSTR, "v1")
	err := d.DictAdd(k1, v1)
	assert.Nil(t, err)
	k1.IncrRefCount()
	v1.IncrRefCount()

	entry = d.DictFind(k1)
	assert.Equal(t, k1, entry.Key)
	assert.Equal(t, v1, entry.Val)

	err = d.DictDelete(k1)
	assert.Nil(t, err)
	entry = d.DictFind(k1)
	assert.Nil(t, entry)
}
