package main

type entry struct {
	key  *RedisObj
	val  *RedisObj
	next *entry
}

type htable struct {
	table []*entry
	size  int64
	mask  int64
	used  int64
}

type DictType interface {
	HashFunc() int
	CompareFunc() int
	KeyDestructor()
	ValDestructor()
}

type Dict struct {
	DictType
	Htable    [2]htable
	rehashidx int
}
