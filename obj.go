package main

import "strconv"

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

func (o *RedisObj) IntVal() int {
	if o.Type_ != REDISSTR {
		return 0
	}
	val, _ := strconv.Atoi(o.Val_.(string))
	return val
}
