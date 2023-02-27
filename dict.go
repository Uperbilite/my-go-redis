package main

import (
	"errors"
	"math"
	"math/rand"
)

const (
	DICT_HT_INITIAL_SIZE     int64 = 8
	DICT_FORCE_RESIZE_RATIO  int64 = 2 // only elements / buckets > 2 can hash table expand.
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

type DictFunc struct {
	HashFunc  func(key *RedisObj) int64
	EqualFunc func(key1, key2 *RedisObj) bool
}

type Dict struct {
	DictFunc
	HashTable [2]*DictHashTable
	rehashIdx int64
	// TODO: impl iterator
}

func DictCreate(dictFunc DictFunc) *Dict {
	var dict Dict
	dict.DictFunc = dictFunc
	dict.rehashIdx = -1
	return &dict
}

func (dict *Dict) DictIsRehashing() bool {
	return dict.rehashIdx != -1
}

func (dict *Dict) DictRehash(step int64) {
	if dict.DictIsRehashing() == false {
		return
	}

	for step > 0 {
		// exchange hash table if rehash is completed.
		if dict.HashTable[0].used == 0 {
			dict.HashTable[0] = dict.HashTable[1]
			dict.HashTable[1] = nil
			dict.rehashIdx = -1
			return
		}

		// find a not null head entry.
		for dict.HashTable[0].table[dict.rehashIdx] == nil {
			dict.rehashIdx += 1
		}

		// rehash all the entry behind the head entry, including head entry.
		var de, nextDe *DictEntry
		de = dict.HashTable[0].table[dict.rehashIdx]
		for de != nil {
			nextDe = de.next
			h := dict.HashFunc(de.Key) & dict.HashTable[1].mask
			de.next = dict.HashTable[1].table[h]
			dict.HashTable[1].table[h] = de
			dict.HashTable[1].used += 1
			dict.HashTable[0].used -= 1
			de = nextDe
		}

		dict.HashTable[0].table[dict.rehashIdx] = nil
		dict.rehashIdx += 1
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
	if dict.DictIsRehashing() || (dict.HashTable[0] != nil && dict.HashTable[0].used > size) {
		return EP_ERR
	}

	var n DictHashTable // the new hash table
	n.size = realSize
	n.mask = realSize - 1
	n.table = make([]*DictEntry, realSize)
	n.used = 0

	// the first initialization.
	if dict.HashTable[0] == nil {
		dict.HashTable[0] = &n
		return nil
	}

	// expand hash table and start rehashing.
	dict.HashTable[1] = &n
	dict.rehashIdx = 0
	return nil
}

func (dict *Dict) dictExpandIfNeeded() error {
	if dict.DictIsRehashing() {
		return nil
	}
	// hash table is empty and expand it to initial size.
	if dict.HashTable[0] == nil {
		return dict.DictExpand(DICT_HT_INITIAL_SIZE)
	}
	if (dict.HashTable[0].used > dict.HashTable[0].size) && (dict.HashTable[0].used/dict.HashTable[0].size > DICT_FORCE_RESIZE_RATIO) {
		return dict.DictExpand(dict.HashTable[0].size * DICT_HT_GROW_RATIO)
	}
	return nil
}

/*
	DictKeyIndex return the index in hash table that the key can
	be inserted into, if the key is already exist, return -1.
*/
func (dict *Dict) DictKeyIndex(key *RedisObj) int64 {
	err := dict.dictExpandIfNeeded()
	if err != nil {
		return -1
	}
	h := dict.HashFunc(key)
	var idx int64
	for i := 0; i <= 1; i++ {
		idx = h & dict.HashTable[i].mask
		he := dict.HashTable[i].table[idx]
		for he != nil {
			if dict.EqualFunc(he.Key, key) {
				return -1
			}
			he = he.next
		}
		if !dict.DictIsRehashing() {
			/*
				if it's in the process of rehashing,
				the index should be in HashTable[1].
			*/
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
	e.Key = key
	e.Key.IncrRefCount()
	e.next = ht.table[idx]
	ht.table[idx] = &e
	ht.used += 1

	return &e
}

func (dict *Dict) DictFind(key *RedisObj) *DictEntry {
	if dict.HashTable[0] == nil {
		return nil
	}
	if dict.DictIsRehashing() {
		dict.DictRehashStep()
	}
	h := dict.HashFunc(key)
	for i := 0; i <= 1; i++ {
		idx := h & dict.HashTable[i].mask
		he := dict.HashTable[i].table[idx]
		for he != nil {
			if dict.EqualFunc(he.Key, key) {
				return he
			}
			he = he.next
		}
		if !dict.DictIsRehashing() {
			// if rehashing, then find key in two hash tables.
			break
		}
	}
	return nil
}

// DictAdd add a new kv pair, return err if key exists.
func (dict *Dict) DictAdd(key, val *RedisObj) error {
	entry := dict.DictAddRaw(key)
	if entry == nil {
		return EX_ERR
	}
	entry.Val = val
	entry.Val.IncrRefCount()
	return nil
}

// DictSet add a new kv pair, or update the exists pair.
func (dict *Dict) DictSet(key, val *RedisObj) {
	if err := dict.DictAdd(key, val); err == nil {
		return
	}
	entry := dict.DictFind(key)
	entry.Val.DecrRefCount()
	entry.Val = val
	val.IncrRefCount()
}

func freeEntry(e *DictEntry) {
	e.Key.DecrRefCount()
	e.Val.DecrRefCount()
}

func (dict *Dict) DictDelete(key *RedisObj) error {
	if dict.HashTable[0] == nil {
		return NK_ERR
	}
	if dict.DictIsRehashing() {
		dict.DictRehashStep()
	}
	h := dict.HashFunc(key)
	for i := 0; i <= 1; i++ {
		idx := h & dict.HashTable[i].mask
		he := dict.HashTable[i].table[idx]
		var prevHe *DictEntry
		for he != nil {
			if dict.EqualFunc(he.Key, key) {
				if prevHe == nil {
					dict.HashTable[i].table[idx] = he.next
				} else {
					prevHe.next = he.next
				}
				freeEntry(he)
				dict.HashTable[i].used -= 1
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

func (dict *Dict) DictGet(key *RedisObj) *RedisObj {
	entry := dict.DictFind(key)
	if entry == nil {
		return nil
	}
	return entry.Val
}

func (dict *Dict) DictGetRandomKey() *DictEntry {
	if dict.HashTable[0] == nil {
		return nil
	}

	if dict.DictIsRehashing() {
		dict.DictRehashStep()
	}

	// get random hash entry.
	var he *DictEntry
	if dict.DictIsRehashing() {
		for he == nil {
			h := rand.Int63n(dict.HashTable[0].size + dict.HashTable[1].size)
			if h >= dict.HashTable[0].size {
				he = dict.HashTable[1].table[h-dict.HashTable[0].size]
			} else {
				he = dict.HashTable[0].table[h]
			}
		}
	} else {
		for he == nil {
			h := rand.Int63n(dict.HashTable[0].size)
			he = dict.HashTable[0].table[h]
		}
	}

	var listLen int64
	origHe := he
	for he != nil {
		he = he.next
		listLen += 1
	}
	listIdx := rand.Int63n(listLen)
	he = origHe
	for listIdx > 0 {
		he = he.next
		listIdx -= 1
	}

	return he
}
