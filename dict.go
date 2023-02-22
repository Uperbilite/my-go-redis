package main

type entry struct {
	key  *interface{}
	val  *interface{}
	next *entry
}

type hashTable struct {
	table []*entry
	size  int64
	mask  int64
	used  int64
}

type DictType struct {
	HashFunction func(key interface{}) int
	KeyCompare   func(key1, key2 interface{}) bool
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
