package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func WriteProc(eventLoop *AeEventLoop, fd int, clientData interface{}) {
	buf := clientData.([]byte)
	n, err := Write(fd, buf)
	if err != nil {
		fmt.Printf("write err: %v\n", err)
		return
	}
	fmt.Printf("write %v bytes\n", n)
	eventLoop.AeDeleteFileEvent(fd, AE_WRITABLE)
}

func ReadProc(eventLoop *AeEventLoop, fd int, clientData interface{}) {
	buf := make([]byte, 10)
	n, err := Read(fd, buf)
	if err != nil {
		fmt.Printf("read err: %v\n", err)
		return
	}
	fmt.Printf("read %v bytes\n", n)
	eventLoop.AeCreateFileEvent(fd, AE_WRITABLE, WriteProc, buf)
}

func AcceptProc(eventLoop *AeEventLoop, fd int, clientData interface{}) {
	cfd, err := Accept(fd)
	if err != nil {
		fmt.Printf("accept err: %v\n", err)
		return
	}
	eventLoop.AeCreateFileEvent(cfd, AE_READABLE, ReadProc, nil)
}

func OnceProc(eventLoop *AeEventLoop, id int, clientData interface{}) {
	t := clientData.(*testing.T)
	assert.Equal(t, 1, id)
	fmt.Printf("time event %v done\n", id)
}

func NormalProc(eventLoop *AeEventLoop, id int, clientData interface{}) {
	end := clientData.(chan struct{})
	fmt.Printf("time event %v done\n", id)
	end <- struct{}{}
}

func TestAe(t *testing.T) {
	aeLoop, err := AeCreateEventLoop()
	assert.Nil(t, err)
	sfd, err := TcpServer(6379, "127.0.0.1")
	aeLoop.AeCreateFileEvent(sfd, AE_READABLE, AcceptProc, nil)
	go aeLoop.AeMain()
	host := [4]byte{127, 0, 0, 1}
	cfd, err := Connect(host, 6379)
	assert.Nil(t, err)
	msg := "helloworld"
	n, err := Write(cfd, []byte(msg))
	assert.Nil(t, err)
	assert.Equal(t, 10, n)
	buf := make([]byte, 10)
	n, err = Read(cfd, buf)
	assert.Nil(t, err)
	assert.Equal(t, 10, n)
	assert.Equal(t, msg, string(buf))
	aeLoop.AeCreateTimeEvent(AE_ONCE, 10, OnceProc, t)
	end := make(chan struct{}, 2)
	aeLoop.AeCreateTimeEvent(AE_NORMAL, 10, NormalProc, end)
	<-end
	<-end
	aeLoop.stop = true
}
