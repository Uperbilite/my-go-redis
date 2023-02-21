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

type DictType interface {
	HashFunc() int
	CompareFunc() int
}

type Dict struct {
	DictType
	HashTable [2]hashTable
	rehashidx int
}

func DictCreate() *Dict {
	// TODO
	return nil
}
