package main

type DictEntry struct {
	key  *RedisObj
	val  *RedisObj
	next *DictEntry
}

type DictHashTable struct {
	table []*DictEntry
	size  int64
	mask  int64
	used  int64
}

type DictType struct {
	HashFunction func(key *RedisObj) int64
	KeyCompare   func(key1, key2 *RedisObj) bool
}

type Dict struct {
	DictType
	HashTable [2]DictHashTable
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

func dictNextPower(size int64) int64 {
	return 0
}

func (dict *Dict) dictExpandIfNeeded() {

}

func (dict *Dict) DictExpand(size int64) {

}

func (dict *Dict) DictGetRandomKey() (key, val *RedisObj) {
	// TODO: get a random item in dict.
	return nil, nil
}

func (dict *Dict) DictDeleteKey(key *RedisObj) {
	// TODO
}
