package main

type ListNode struct {
	Val  *RedisObj
	next *ListNode
	prev *ListNode
}

type ListType struct {
	EqualFunc func(a, b *RedisObj) bool
}

type List struct {
	ListType
	head   *ListNode
	tail   *ListNode
	length int
}

func ListCreate(listType ListType) *List {
	var list List
	list.ListType = listType
	return &list
}

func (list *List) ListLength() int {
	return list.length
}

func (list *List) ListFirst() *ListNode {
	return list.head
}

func (list *List) ListLast() *ListNode {
	return list.tail
}

func (list *List) ListSearchKey(val *RedisObj) *ListNode {
	p := list.head
	for p != nil {
		if list.EqualFunc(p.Val, val) {
			break
		}
		p = p.next
	}
	return p
}

func (list *List) ListAddNodeHead(val *RedisObj) {
	var node ListNode
	node.Val = val
	if list.head == nil {
		list.head = &node
		list.tail = &node
	} else {
		node.next = list.head
		list.head.prev = &node
		list.head = &node
	}
	list.length += 1
}

func (list *List) ListAddNodeTail(val *RedisObj) {
	var node ListNode
	node.Val = val
	if list.head == nil {
		list.head = &node
		list.tail = &node
	} else {
		node.prev = list.tail
		list.tail.next = &node
		list.tail = &node
	}
	list.length += 1
}

func (list *List) ListDelKey(val *RedisObj) {
	p := list.ListSearchKey(val)
	if list.head == p {
		p.next.prev = nil
		list.head = p.next
		p.next = nil
	} else if list.tail == p {
		p.prev.next = nil
		list.tail = p.prev
		p.prev = nil
	} else {
		p.prev.next = p.next
		p.next.prev = p.prev
		p.next = nil
		p.prev = nil
	}
	list.length -= 1
}

func (list *List) ListDelNode(n *ListNode) {
	list.ListDelKey(n.Val)
}
