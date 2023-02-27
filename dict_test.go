package main

import (
	"fmt"
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
	assert.Equal(t, 2, k1.refcount)
	assert.Equal(t, 2, v1.refcount)

	entry = d.DictFind(k1)
	assert.Equal(t, k1, entry.Key)
	assert.Equal(t, v1, entry.Val)

	err = d.DictDelete(k1)
	assert.Nil(t, err)
	entry = d.DictFind(k1)
	assert.Nil(t, entry)
	assert.Equal(t, 1, k1.refcount)
	assert.Equal(t, 1, v1.refcount)
}

func TestRehash(t *testing.T) {
	d := DictCreate(DictType{
		HashFunction: RedisStrHash,
		KeyCompare:   RedisStrEqual,
	})
	entry := d.DictGetRandomKey()
	assert.Nil(t, entry)

	size := int(DICT_HT_INITIAL_SIZE * (DICT_FORCE_RESIZE_RATIO + 1))
	for i := 0; i < size; i++ {
		key := CreateObject(REDISSTR, fmt.Sprintf("k%v", i))
		val := CreateObject(REDISSTR, fmt.Sprintf("v%v", i))
		err := d.DictAdd(key, val)
		assert.Nil(t, err)
	}
	assert.Equal(t, false, d.DictIsRehashing())

	key := CreateObject(REDISSTR, fmt.Sprintf("k%v", size))
	val := CreateObject(REDISSTR, fmt.Sprintf("v%v", size))
	err := d.DictAdd(key, val)
	assert.Nil(t, err)
	assert.Equal(t, true, d.DictIsRehashing())
	assert.Equal(t, int64(0), d.rehashidx)
	assert.Equal(t, DICT_HT_INITIAL_SIZE, d.HashTable[0].size)
	assert.Equal(t, DICT_HT_INITIAL_SIZE*DICT_HT_GROW_RATIO, d.HashTable[1].size)

	for i := 0; i < int(d.HashTable[0].size)+1; i++ {
		d.DictGetRandomKey()
	}
	assert.Equal(t, false, d.DictIsRehashing())
	assert.Equal(t, DICT_HT_INITIAL_SIZE*DICT_HT_GROW_RATIO, d.HashTable[0].size)
	assert.Nil(t, d.HashTable[1])

	for i := 0; i < size+1; i++ {
		key := CreateObject(REDISSTR, fmt.Sprintf("k%v", i))
		entry := d.DictFind(key)
		assert.NotNil(t, entry)
		assert.Equal(t, fmt.Sprintf("v%v", i), entry.Val.StrVal())
	}

}
