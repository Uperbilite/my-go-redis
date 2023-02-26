package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
	"testing"
)

func EchoServer(end chan struct{}) {
	sfd, err := TcpServer(6379, "127.0.0.1")
	if err != nil {
		fmt.Printf("tcp server error: %v\n", err)
	}
	end <- struct{}{}
	cfd, err := Accept(sfd)
	if err != nil {
		fmt.Printf("server accept error: %v\n", err)
	}
	buf := make([]byte, 10)
	n, err := unix.Read(cfd, buf)
	fmt.Printf("server read %v bytes\n", n)
	n, err = unix.Write(cfd, buf)
	if err != nil {
		fmt.Printf("server write error: %v\n", err)
	}
	fmt.Printf("server write %v bytes\n", n)
	<-end
}

func TestNet(t *testing.T) {
	end := make(chan struct{})
	go EchoServer(end)
	<-end
	host := [4]byte{127, 0, 0, 1}
	cfd, err := Connect(host, 6379)
	assert.Nil(t, err)
	msg := "helloworld"
	n, err := unix.Write(cfd, []byte(msg))
	assert.Nil(t, err)
	assert.Equal(t, 10, n)
	buf := make([]byte, 10)
	n, err = unix.Read(cfd, buf)
	assert.Nil(t, err)
	assert.Equal(t, 10, n)
	assert.Equal(t, msg, string(buf))
	end <- struct{}{}
}
