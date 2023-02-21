package main

type Node struct {
	val  *RedisObj
	next *Node
	prev *Node
}

type List struct {
	head *Node
	tail *Node
}

func ListCreate() *List {
	// TODO
	return nil
}
