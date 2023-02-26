package main

import (
	"errors"
	"math"
)

const (
	DICT_HT_INITIAL_SIZE     int64 = 8
	DICT_FORCE_RESIZE_RATIO  int64 = 3
	DICT_HT_GROW_RATIO       int64 = 2
	DICT_DEFAULT_REHASH_STEP int64 = 1
)

var (
	EP_ERR = errors.New("expand error")
	EX_ERR = errors.New("key exists error")
)

type DictEntry struct {
	Key  *RedisObj
	Val  *RedisObj
	next *DictEntry
}

type DictHashTable struct {
	table []*DictEntry
	size  int64
	mask  int64 // size mask
	used  int64
}

type DictType struct {
	HashFunction func(key *RedisObj) int64
	KeyCompare   func(key1, key2 *RedisObj) bool
}

type Dict struct {
	DictType
	HashTable [2]*DictHashTable
	rehashidx int64
}

func DictCreate(dictType DictType) *Dict {
	var dict Dict
	dict.DictType = dictType
	return &dict
}

func (dict *Dict) DictIsRehashing() bool {
	return dict.rehashidx != -1
}

func (dict *Dict) DictRehash(step int64) {
	for step > 0 {
		// exchange hash table if rehash is completed.
		if dict.HashTable[0].used == 0 {
			dict.HashTable[0] = dict.HashTable[1]
			dict.HashTable[1] = nil
			dict.rehashidx = -1
			return
		}

		// find a not null head entry.
		for dict.HashTable[0].table[dict.rehashidx] == nil {
			dict.rehashidx += 1
		}

		// hash all the entry behind head entry, including head entry.
		var de, nextDe *DictEntry
		de = dict.HashTable[0].table[dict.rehashidx]
		for de != nil {
			nextDe = de.next
			h := dict.HashFunction(de.Key) & dict.HashTable[1].mask
			de.next = dict.HashTable[1].table[h]
			dict.HashTable[1].table[h] = de
			dict.HashTable[1].used += 1
			dict.HashTable[0].used -= 1
			de = nextDe
		}

		dict.HashTable[0].table[dict.rehashidx] = nil
		dict.rehashidx += 1
		step -= 1
	}
}

func (dict *Dict) DictRehashStep() {
	// TODO: if dict->iterators == 0
	dict.DictRehash(DICT_DEFAULT_REHASH_STEP)
}

func dictNextPower(size int64) int64 {
	for i := DICT_HT_INITIAL_SIZE; i < math.MaxInt64; i *= 2 {
		if i >= size {
			return i
		}
	}
	return -1
}

func (dict *Dict) DictExpand(size int64) error {
	realSize := dictNextPower(size)
	if dict.DictIsRehashing() || dict.HashTable[0].used > size {
		return EP_ERR
	}

	var n DictHashTable
	n.size = realSize
	n.mask = realSize - 1
	n.table = make([]*DictEntry, realSize)
	n.used = 0

	// the first initialization.
	if dict.HashTable[0] == nil {
		dict.HashTable[0] = &n
		return nil
	}

	// expanded hash table.
	dict.HashTable[1] = &n
	dict.rehashidx = 0
	return nil
}

func (dict *Dict) dictExpandIfNeeded() error {
	if dict.DictIsRehashing() {
		return nil
	}
	if dict.HashTable[0].size == 0 {
		return dict.DictExpand(DICT_HT_INITIAL_SIZE)
	}
	if (dict.HashTable[0].used > dict.HashTable[0].size) && (dict.HashTable[0].used/dict.HashTable[0].size > DICT_FORCE_RESIZE_RATIO) {
		return dict.DictExpand(dict.HashTable[0].size * DICT_HT_GROW_RATIO)
	}
	return nil
}

func (dict *Dict) DictKeyIndex(key *RedisObj) int64 {
	err := dict.dictExpandIfNeeded()
	if err != nil {
		return -1
	}
	h := dict.HashFunction(key)
	var idx int64
	for i := 0; i <= 1; i++ {
		idx = h & dict.HashTable[i].mask
		he := dict.HashTable[i].table[idx]
		for he != nil {
			if dict.KeyCompare(he.Key, key) {
				return -1
			}
			he = he.next
		}
		if !dict.DictIsRehashing() {
			break
		}
	}
	return idx
}

func (dict *Dict) DictAddRaw(key *RedisObj) *DictEntry {
	if dict.DictIsRehashing() {
		dict.DictRehashStep()
	}
	idx := dict.DictKeyIndex(key)
	if idx == -1 {
		return nil
	}

	// add key and return entry
	var ht *DictHashTable
	if dict.DictIsRehashing() {
		ht = dict.HashTable[1]
	} else {
		ht = dict.HashTable[0]
	}

	var e DictEntry
	e.next = ht.table[idx]
	ht.table[idx] = &e
	ht.used += 1
	return &e
}

func (dict *Dict) DictAdd(key, val *RedisObj) error {
	entry := dict.DictAddRaw(key)
	if entry == nil {
		return EX_ERR
	}
	entry.Val = val
	return nil
}

func (dict *Dict) DictGetRandomKey() (key, val *RedisObj) {
	// TODO: get a random item in dict.
	return nil, nil
}

func (dict *Dict) DictDeleteKey(key *RedisObj) {
	// TODO
}
