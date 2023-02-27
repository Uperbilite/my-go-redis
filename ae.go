package main

import (
	"golang.org/x/sys/unix"
	"log"
	"time"
)

type FeType int

const (
	AE_READABLE FeType = 1
	AE_WRITABLE FeType = 2
)

type TeType int

const (
	AE_NORMAL TeType = 1 // repeating exec time event.
	AE_ONCE   TeType = 2 // exec time event once.
)

type AeFileProc func(eventLoop *AeEventLoop, fd int, clientData interface{})
type AeTimeProc func(eventLoop *AeEventLoop, id int, clientData interface{})

type AeFileEvent struct {
	fd         int
	mask       FeType
	fileProc   AeFileProc
	clientData interface{}
}

type AeTimeEvent struct {
	id         int
	mask       TeType
	when       int64 // ms
	duration   int64 // ms
	timeProc   AeTimeProc
	clientData interface{}
	next       *AeTimeEvent
}

type AeEventLoop struct {
	FileEvents      map[int]*AeFileEvent
	TimeEventHead   *AeTimeEvent
	epfd            int
	timeEventNextId int
	stop            bool
}

func GetMsTime() int64 {
	return time.Now().UnixMilli()
}

func getFeKey(fd int, mask FeType) int {
	if mask == AE_READABLE {
		return fd
	} else {
		return fd * -1
	}
}

var fe2ep [3]uint32 = [3]uint32{0, unix.EPOLLIN, unix.EPOLLOUT}

// getEpollMask get epoll event type by file event.
func (eventLoop *AeEventLoop) getEpollMask(fd int) uint32 {
	var ev uint32
	if eventLoop.FileEvents[getFeKey(fd, AE_READABLE)] != nil {
		ev |= fe2ep[AE_READABLE]
	}
	if eventLoop.FileEvents[getFeKey(fd, AE_WRITABLE)] != nil {
		ev |= fe2ep[AE_WRITABLE]
	}
	return ev
}

func AeCreateEventLoop() (*AeEventLoop, error) {
	epfd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	return &AeEventLoop{
		FileEvents:      make(map[int]*AeFileEvent),
		epfd:            epfd,
		timeEventNextId: 1,
		stop:            false,
	}, nil
}

// AeCreateFileEvent Create a file event and insert into the head of file event list.
func (eventLoop *AeEventLoop) AeCreateFileEvent(fd int, mask FeType, proc AeFileProc, clientData interface{}) {
	// epoll ctl
	ev := eventLoop.getEpollMask(fd)
	if ev&fe2ep[mask] != 0 {
		return
	}
	op := unix.EPOLL_CTL_ADD
	if ev != 0 {
		op = unix.EPOLL_CTL_MOD
	}
	ev |= fe2ep[mask]
	err := unix.EpollCtl(eventLoop.epfd, op, fd, &unix.EpollEvent{
		Events: ev,
		Fd:     int32(fd),
		Pad:    0,
	})
	if err != nil {
		log.Printf("epoll ctl err: %v\n", err)
		return
	}

	// ae ctl
	var fe AeFileEvent
	fe.fd = fd
	fe.mask = mask
	fe.fileProc = proc
	fe.clientData = clientData
	eventLoop.FileEvents[getFeKey(fd, mask)] = &fe
	log.Printf("ae crearte file event fd:%v, mask:%v\n", fd, mask)
}

// AeDeleteFileEvent Delete file event by iterating file event list.
func (eventLoop *AeEventLoop) AeDeleteFileEvent(fd int, mask FeType) {
	// epoll ctl
	op := unix.EPOLL_CTL_DEL
	ev := eventLoop.getEpollMask(fd)
	/*
		Get events except the event which is mapped from file event
		type. If there are events left, then modify this epoll event.
		Otherwise, delete this epoll event.
	*/
	ev &= ^fe2ep[mask]
	if ev != 0 {
		op = unix.EPOLL_CTL_MOD
	}
	err := unix.EpollCtl(eventLoop.epfd, op, fd, &unix.EpollEvent{
		Events: ev,
		Fd:     int32(fd),
		Pad:    0,
	})
	if err != nil {
		log.Printf("epoll del err: %v\n", err)
		return
	}

	// ae ctl
	eventLoop.FileEvents[getFeKey(fd, mask)] = nil
	log.Printf("ae delete file event fd:%v, mask:%v\n", fd, mask)
}

// AeCreateTimeEvent Create time event and insert into the head of time event list.
func (eventLoop *AeEventLoop) AeCreateTimeEvent(mask TeType, duration int64, proc AeTimeProc, clientData interface{}) int {
	id := eventLoop.timeEventNextId
	eventLoop.timeEventNextId++
	var te AeTimeEvent
	te.id = id
	te.mask = mask
	te.duration = duration
	te.when = GetMsTime() + duration
	te.timeProc = proc
	te.clientData = clientData
	te.next = eventLoop.TimeEventHead
	eventLoop.TimeEventHead = &te
	return id
}

// AeDeleteTimeEvent Delete time event by id.
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

func (eventLoop *AeEventLoop) AeProcessEvents(tes []*AeTimeEvent, fes []*AeFileEvent) {
	for _, te := range tes {
		te.timeProc(eventLoop, te.id, te.clientData)
		switch te.mask {
		case AE_NORMAL:
			/*
				AE_NORMAL means this event is repeating event, the
				next time of this event's operation is updated.
			*/
			te.when = GetMsTime() + te.duration
		case AE_ONCE:
			eventLoop.AeDeleteTimeEvent(te.id)
		}
	}
	if len(fes) > 0 {
		log.Println("ae is processing file events")
		for _, fe := range fes {
			fe.fileProc(eventLoop, fe.fd, fe.clientData)
		}
	}
}

// nearestTime get the nearest time of the next time event.
func (eventLoop *AeEventLoop) nearestTime() int64 {
	nearest := GetMsTime() + 1000
	te := eventLoop.TimeEventHead
	for te != nil {
		if te.when < nearest {
			nearest = te.when
		}
		te = te.next
	}
	return nearest
}

func (eventLoop *AeEventLoop) AeWait() (tes []*AeTimeEvent, fes []*AeFileEvent) {
	// TODO: error handle
	timeout := eventLoop.nearestTime() - time.Now().UnixMilli()
	if timeout <= 0 {
		timeout = 10 // at least wait 10ms
	}
	var epollEvents [128]unix.EpollEvent
	n, err := unix.EpollWait(eventLoop.epfd, epollEvents[:], int(timeout))
	if err != nil {
		log.Printf("epoll wait warning: %v\n", err)
	}
	if n > 0 {
		log.Printf("ae get %v epoll events\n", n)
	}

	// collect file event which is ready
	for i := 0; i < n; i++ {
		if epollEvents[i].Events&unix.EPOLLIN != 0 {
			fe := eventLoop.FileEvents[getFeKey(int(epollEvents[i].Fd), AE_READABLE)]
			if fe != nil {
				fes = append(fes, fe)
			}
		}
		if epollEvents[i].Events&unix.EPOLLOUT != 0 {
			fe := eventLoop.FileEvents[getFeKey(int(epollEvents[i].Fd), AE_WRITABLE)]
			if fe != nil {
				fes = append(fes, fe)
			}
		}
	}

	// collect time event which is ready
	now := GetMsTime()
	te := eventLoop.TimeEventHead
	for te != nil {
		if te.when < now {
			tes = append(tes, te)
		}
		te = te.next
	}

	return
}

func (eventLoop *AeEventLoop) AeMain() {
	eventLoop.stop = false
	for eventLoop.stop != true {
		tes, fes := eventLoop.AeWait()
		eventLoop.AeProcessEvents(tes, fes)
	}
}
