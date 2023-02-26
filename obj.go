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

func (o *RedisObj) StrVal() string {
	if o.Type_ != REDISSTR {
		return ""
	}
	return o.Val_.(string)
}

func CreateFromInt(val int) *RedisObj {
	return &RedisObj{
		Type_:    REDISSTR,
		Val_:     strconv.Itoa(val),
		refcount: 1,
	}
}

func CreateObject(t RedisType, ptr interface{}) *RedisObj {
	return &RedisObj{
		Type_:    t,
		Val_:     ptr,
		refcount: 1,
	}
}

func (o *RedisObj) IncrRefCount() {
	o.refcount++
}

func (o *RedisObj) DecrRefCount() {
	o.refcount--
	if o.refcount == 0 {
		o.Val_ = nil
	}
}
