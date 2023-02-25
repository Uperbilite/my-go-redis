package main

type entry struct {
	key  *RedisObj
	val  *RedisObj
	next *entry
}

type hashTable struct {
	table []*entry
	size  int64
	mask  int64
	used  int64
}

type DictType struct {
	HashFunction func(key *RedisObj) int
	KeyCompare   func(key1, key2 *RedisObj) bool
}

type Dict struct {
	DictType
	HashTable [2]hashTable
	rehashidx int
}

func DictCreate(dictType DictType) *Dict {
	var dict Dict
	dict.DictType = dictType
	return &dict
}

func (dict *Dict) RandomGet() (key, val *RedisObj) {
	// TODO: get a random item in dict.
	return nil, nil
}

func (dict *Dict) DeleteKey(key *RedisObj) {
	// TODO
}
