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

type FileProc func(loop *AeLoop, fd int, clientData interface{}, mask FeType)
type TimeProc func(loop *AeLoop, id int, clientData interface{}) int

type AeFileEvent struct {
	fd         int
	mask       FeType
	proc       FileProc
	clientData interface{}
	next       *AeFileEvent
}

type AeTimeEvent struct {
	id         int
	mask       TeType
	when       int64
	proc       TimeProc
	clientData interface{}
	next       *AeTimeEvent
}

type AeLoop struct {
	FileEvents      *AeFileEvent
	TimeEvents      *AeTimeEvent
	timeEventNextId int
	stop            int
}

func AeCreateLoop() *AeLoop {
	// TODO
	return nil
}

func (loop *AeLoop) AeMain() {
	// TODO
}
