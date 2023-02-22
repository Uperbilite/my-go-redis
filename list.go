package main

type ListNode struct {
	val  interface{}
	next *ListNode
	prev *ListNode
}

type ListType struct {
	EqualFunc func(a, b interface{}) bool
}

type List struct {
	ListType
	head *ListNode
	tail *ListNode
}

func ListCreate(listType ListType) *List {
	var list List
	list.ListType = listType
	return &list
}

func (list *List) ListAddNodeHead(val interface{}) {
	var node ListNode
	node.val = val
	if list.head == nil {
		list.head = &node
		list.tail = &node
	} else {
		node.prev = list.tail
		list.tail.next = &node
		list.tail = list.tail.next
	}
}

func (list *List) Remove(val interface{}) {
	p := list.head
	for p != nil {
		if list.EqualFunc(p.val, val) {
			break
		}
		p = p.next
	}
	if p != nil {
		p.prev = p.next
		if p.next != nil {
			p.next.prev = p.prev
		}
		p.prev = nil
		p.next = nil
	}

}
