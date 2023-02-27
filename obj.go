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
	refCount int
}

func (o *RedisObj) IntVal() int64 {
	if o.Type_ != REDISSTR {
		return 0
	}
	val, _ := strconv.ParseInt(o.Val_.(string), 10, 64)
	return val
}

func (o *RedisObj) StrVal() string {
	if o.Type_ != REDISSTR {
		return ""
	}
	return o.Val_.(string)
}

func CreateFromInt(val int64) *RedisObj {
	return &RedisObj{
		Type_:    REDISSTR,
		Val_:     strconv.FormatInt(val, 10),
		refCount: 1,
	}
}

func CreateObject(t RedisType, v interface{}) *RedisObj {
	return &RedisObj{
		Type_:    t,
		Val_:     v,
		refCount: 1,
	}
}

func (o *RedisObj) IncrRefCount() {
	o.refCount++
}

func (o *RedisObj) DecrRefCount() {
	o.refCount--
	if o.refCount == 0 {
		// let GC clear the object.
		o.Val_ = nil
	}
}
