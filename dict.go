package main

import (
	"errors"
	"math"
	"math/rand"
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
	NK_ERR = errors.New("key doesn't exist error")
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

func (dict *Dict) DictFind(key *RedisObj) *DictEntry {
	if dict.HashTable[0].size == 0 {
		return nil
	}
	if dict.DictIsRehashing() {
		dict.DictRehashStep()
	}
	h := dict.HashFunction(key)
	for i := 0; i <= 1; i++ {
		idx := h & dict.HashTable[i].mask
		e := dict.HashTable[i].table[idx]
		if e != nil {
			if dict.KeyCompare(e.Key, key) {
				return e
			}
			e = e.next
		}
		if !dict.DictIsRehashing() {
			break
		}
	}
	return nil
}

func freeEntry(e *DictEntry) {
	e.Key.DecrRefCount()
	e.Val.DecrRefCount()
}

func (dict *Dict) DictDelete(key *RedisObj) error {
	if dict.HashTable[0].size == 0 {
		return NK_ERR
	}
	if dict.DictIsRehashing() {
		dict.DictRehashStep()
	}
	h := dict.HashFunction(key)
	for i := 0; i <= 1; i++ {
		idx := h & dict.HashTable[i].mask
		he := dict.HashTable[i].table[idx]
		var prevHe *DictEntry
		for he != nil {
			if dict.KeyCompare(he.Key, key) {
				if prevHe == nil {
					dict.HashTable[i].table[idx] = he.next
				} else {
					prevHe.next = he.next
				}
				freeEntry(he)
				return nil
			}
			prevHe = he
			he = he.next
		}
		if !dict.DictIsRehashing() {
			break
		}
	}
	return NK_ERR
}

func (dict *Dict) DictGetRandomKey() *DictEntry {
	if dict.HashTable[0].size == 0 || dict.HashTable[0].used == 0 {
		return nil
	}
	t := 0
	if dict.DictIsRehashing() {
		dict.DictRehashStep()
		if dict.HashTable[1].used > dict.HashTable[0].used {
			t = 1
		}
	}
	idx := rand.Int63n(dict.HashTable[t].size)
	cnt := 0
	for dict.HashTable[t].table[idx] == nil && cnt < 1000 {
		idx = rand.Int63n(dict.HashTable[t].size)
		cnt += 1
	}
	if dict.HashTable[t].table[idx] == nil {
		return nil
	}

	var listLen int64
	p := dict.HashTable[t].table[idx]
	for p != nil {
		listLen += 1
		p = p.next
	}
	listIdx := rand.Int63n(listLen)
	p = dict.HashTable[t].table[idx]
	for i := int64(0); i < listIdx; i++ {
		p = p.next
	}
	return p
}
