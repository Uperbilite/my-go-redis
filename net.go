package main

import (
	"golang.org/x/sys/unix"
	"log"
	"strconv"
	"strings"
)

const BACKLOG = 64

func Accept(fd int) (int, error) {
	nfd, _, err := unix.Accept(fd)
	// ignore client addr for now
	return nfd, err
}

func Connect(host [4]byte, port int) (int, error) {
	s, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
	if err != nil {
		log.Printf("init socket err: %v\n", err)
		return -1, err
	}
	var sockAddr unix.SockaddrInet4
	sockAddr.Addr = host
	sockAddr.Port = port
	err = unix.Connect(s, &sockAddr)
	if err != nil {
		log.Printf("connect err: %v\n", err)
		return -1, err
	}
	return s, nil
}

func addrInet4ToBytes(addr string) ([4]byte, error) {
	var result [4]byte
	as := strings.Split(addr, ".")
	for i := 0; i < 4; i++ {
		a, err := strconv.Atoi(as[i])
		if err != nil {
			return [4]byte{}, err
		}
		result[i] = byte(a)
	}
	return result, nil
}

func TcpServer(port int, addr string) (int, error) {
	s, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
	if err != nil {
		log.Printf("init socket err: %v\n", err)
		return -1, nil
	}

	err = unix.SetsockoptInt(s, unix.SOL_SOCKET, unix.SO_REUSEPORT, port)
	if err != nil {
		log.Printf("set SO_REUSEPORT err: %v\n", err)
		unix.Close(s)
		return -1, nil
	}

	var sockAddr unix.SockaddrInet4
	sockAddr.Port = port
	sockAddr.Addr, err = addrInet4ToBytes(addr)
	if err != nil {
		log.Printf("invalid server addr: %v\n", addr)
		unix.Close(s)
		return -1, nil
	}

	err = unix.Bind(s, &sockAddr)
	if err != nil {
		log.Printf("bind addr err: %v\n", err)
		unix.Close(s)
		return -1, nil
	}

	err = unix.Listen(s, BACKLOG)
	if err != nil {
		log.Printf("listen socket err: %v\n", err)
		unix.Close(s)
		return -1, nil
	}

	return s, nil
}
