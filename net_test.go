package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func EchoServer(start, end chan struct{}) {
	sfd, err := TcpServer(6379, "127.0.0.1")
	if err != nil {
		fmt.Printf("tcp server error: %v\n", err)
	}
	fmt.Println("server started")
	start <- struct{}{}
	cfd, err := Accept(sfd)
	if err != nil {
		fmt.Printf("server accept error: %v\n", err)
	}
	buf := make([]byte, 10)
	n, err := Read(cfd, buf)
	fmt.Printf("server read %v bytes\n", n)
	n, err = Write(cfd, buf)
	if err != nil {
		fmt.Printf("server write error: %v\n", err)
	}
	fmt.Printf("server write %v bytes\n", n)
	end <- struct{}{}
}

func TestNet(t *testing.T) {
	start := make(chan struct{})
	end := make(chan struct{})
	go EchoServer(start, end)
	<-start
	host := [4]byte{127, 0, 0, 1}
	cfd, err := Connect(host, 6379)
	assert.Nil(t, err)
	msg := "helloworld"
	n, err := Write(cfd, []byte(msg))
	assert.Nil(t, err)
	assert.Equal(t, 10, n)
	<-end
	buf := make([]byte, 10)
	n, err = Read(cfd, buf)
	assert.Nil(t, err)
	assert.Equal(t, 10, n)
	assert.Equal(t, msg, string(buf))
}
