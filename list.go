package main

type ListNode struct {
	Val  *RedisObj
	next *ListNode
	prev *ListNode
}

type ListFunc struct {
	EqualFunc func(a, b *RedisObj) bool
}

type List struct {
	ListFunc
	head   *ListNode
	tail   *ListNode
	length int
}

func ListCreate(listFunc ListFunc) *List {
	var list List
	list.ListFunc = listFunc
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
	if list.length == 0 {
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
	if list.length == 0 {
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
		if p.next != nil {
			p.next.prev = nil
		}
		list.head = p.next
		p.next = nil
	}

	if list.tail == p {
		if p.prev != nil {
			p.prev.next = nil
		}
		list.tail = p.prev
		p.prev = nil
	}

	if p.prev != nil {
		p.prev.next = p.next
	}
	if p.next != nil {
		p.next.prev = p.prev
	}
	p.next = nil
	p.prev = nil

	list.length -= 1
}

func (list *List) ListDelNode(n *ListNode) {
	if n != nil {
		list.ListDelKey(n.Val)
	}
}
