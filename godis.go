package main

import "fmt"

type RedisDB struct {
	data   *Dict
	expire *Dict
}

type RedisServer struct {
	fd      int
	port    int
	db      *RedisDB
	clients *List
	aeLoop  *AeLoop
}

type RedisClient struct {
	fd    int
	db    *RedisDB
	query string
	reply *List
}

func main() {
	fmt.Println("Hello Redis!")
}
