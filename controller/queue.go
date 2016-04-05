package controller

import "time"

type QueueItem struct {
	Req   **Request
	Time  time.Time
	index int
}

type Queue []*QueueItem

func NewQueue(size int) Queue {
	return make([]*QueueItem, size)
}

func (q Queue) Less(i, j int) bool {
	return q[i].Time.Before(q[j].Time)
}

func (q Queue) Len() int {
	return len(q)
}

func (q Queue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *Queue) Push(x interface{}) {
	n := len(*q)
	item := x.(*QueueItem)
	item.index = n
	*q = append(*q, item)
}
