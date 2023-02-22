package main

type RedisType uint8
type RedisVal interface{}

const (
	REDISSTR  RedisType = 0x01
	REDISLIST RedisType = 0x02
	REDISDICT RedisType = 0x03
)

type RedisObj struct {
	Type_    RedisType
	Val_     RedisVal
	refcount int
}
