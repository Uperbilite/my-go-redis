package main

type FeType int

const (
	AE_READABLE FeType = 1
	AE_WRITABLE FeType = 2
)

type TeType int

const (
	AE_NORMAL TeType = 1
	AE_ONCE   TeType = 2
)

type aeFileProc func(eventLoop *AeEventLoop, fd int, clientData interface{})
type aeTimeProc func(eventLoop *AeEventLoop, id int, clientData interface{})

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
	when       int64 // sec
	timeProc   aeTimeProc
	clientData interface{}
	next       *AeTimeEvent
}

type AeEventLoop struct {
	timeEventNextId int
	FileEventHead   *AeFileEvent
	TimeEventHead   *AeTimeEvent
	stop            bool
}

func AeCreateEventLoop() *AeEventLoop {
	var eventLoop AeEventLoop
	eventLoop.timeEventNextId = 1
	eventLoop.stop = false
	return &eventLoop
}

// AeCreateFileEvent Create a file event and insert into the head of file event list.
func (eventLoop *AeEventLoop) AeCreateFileEvent(fd int, mask FeType, proc aeFileProc, clientData interface{}) {
	var fe AeFileEvent
	fe.fd = fd
	fe.mask = mask
	fe.fileProc = proc
	fe.clientData = clientData
	fe.next = eventLoop.FileEventHead
	eventLoop.FileEventHead = &fe
}

// AeDeleteFileEvent Delete by iterating file event list.
func (eventLoop *AeEventLoop) AeDeleteFileEvent(fd int, mask FeType) {
	var fe, prev *AeFileEvent
	fe = eventLoop.FileEventHead
	for fe != nil {
		if fe.fd == fd && fe.mask == mask {
			if prev == nil {
				eventLoop.FileEventHead = fe.next
			} else {
				prev.next = fe.next
			}
			fe.next = nil
			break
		}
		prev = fe
		fe = fe.next
	}
}

// AeCreateTimeEvent Create time event and insert into the head of time event list.
func (eventLoop *AeEventLoop) AeCreateTimeEvent(mask TeType, seconds int64, proc aeTimeProc, clientData interface{}) int {
	id := eventLoop.timeEventNextId
	eventLoop.timeEventNextId++
	var te AeTimeEvent
	te.id = id
	te.mask = mask
	te.when = seconds
	te.clientData = clientData
	te.next = eventLoop.TimeEventHead
	eventLoop.TimeEventHead = &te
	return id
}

func (eventLoop *AeEventLoop) AeDeleteTimeEvent(id int) {
	var te, prev *AeTimeEvent
	te = eventLoop.TimeEventHead
	for te != nil {
		if te.id == id {
			if prev == nil {
				eventLoop.TimeEventHead = te.next
			} else {
				prev.next = te.next
			}
			te.next = nil
			break
		}
		prev = te
		te = te.next
	}
}

func (eventLoop *AeEventLoop) AeProcessEvents(tes []AeTimeEvent, fes []AeFileEvent) {
	for _, te := range tes {
		te.timeProc(eventLoop, te.id, te.clientData)
		if te.mask == AE_ONCE {
			eventLoop.AeDeleteTimeEvent(te.id)
		}
	}
	for _, fe := range fes {
		fe.fileProc(eventLoop, fe.fd, fe.clientData)
		eventLoop.AeDeleteFileEvent(fe.fd, fe.mask)
	}
}

func (eventLoop *AeEventLoop) AeWait() (tes []AeTimeEvent, fes []AeFileEvent) {
	// TODO: search time && epoll wait
	return nil, nil
}

func (eventLoop *AeEventLoop) AeMain() {
	eventLoop.stop = false
	for eventLoop.stop != true {
		tes, fes := eventLoop.AeWait()
		eventLoop.AeProcessEvents(tes, fes)
	}
}
