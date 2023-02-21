package main

type FeType int

const (
	AE_READABLE FeType = 1
	AE_WRITABLE FeType = 2
)

type TeType int

const (
	AE_NORMAL TeType = 1
	AE_ONCE   TeType = 1
)

type aeFileProc func(eventLoop *AeEventLoop, fd int, clientData interface{}, mask FeType)
type aeTimeProc func(eventLoop *AeEventLoop, id int, clientData interface{}) int

type AeFileEvent struct {
	fd         int
	mask       FeType
	fileProc   aeFileProc
	clientData interface{}
	next       *AeFileEvent
}

type AeTimeEvent struct {
	id         int
	mask       TeType
	when       int64
	timeProc   aeTimeProc
	clientData interface{}
	next       *AeTimeEvent
}

type AeEventLoop struct {
	timeEventNextId int
	FileEventHead   *AeFileEvent
	TimeEventHead   *AeTimeEvent
	stop            int
}

func AeCreateEventLoop() *AeEventLoop {
	// TODO
	return nil
}

func (eventLoop *AeEventLoop) AeMain() {
	// TODO
}
