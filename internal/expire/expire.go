package expire

import (
	"sync"
	"time"
)

// Item is a something that can expire
type Item interface {
	Expires() time.Time
}

// List of expireable items
type List struct {
	mu      sync.Mutex
	queue   queue
	bgrun   bool
	Expired func(item Item)
}

// Push an item onto the queue
func (list *List) Push(item Item) {
	unix := item.Expires().UnixNano()
	list.mu.Lock()
	if !list.bgrun {
		list.bgrun = true
		go list.bg()
	}
	list.queue.push(unix, item)
	list.mu.Unlock()
}

func (list *List) bg() {
	now := time.Now().UnixNano()
	for {
		list.mu.Lock()
		if list.queue.len == 0 {
			list.bgrun = false
			list.mu.Unlock()
			break
		}
		if now > list.queue.peek().unix { // now.After(list.queue.peek().unix)
			n := list.queue.pop()
			exfn := list.Expired
			list.mu.Unlock()
			if exfn != nil {
				exfn(n.item)
			}
		} else {
			list.mu.Unlock()
			time.Sleep(time.Second / 10)
			now = time.Now().UnixNano()
		}
	}
}

type qnode struct {
	unix int64
	item Item
}

type queue struct {
	nodes []qnode
	len   int
}

func (q *queue) push(unix int64, item Item) {
	if q.nodes == nil {
		q.nodes = make([]qnode, 2)
	} else {
		q.nodes = append(q.nodes, qnode{})
	}
	i := q.len + 1
	j := i / 2
	for i > 1 && q.nodes[j].unix > unix {
		q.nodes[i] = q.nodes[j]
		i = j
		j = j / 2
	}
	q.nodes[i].unix = unix
	q.nodes[i].item = item
	q.len++
}

func (q *queue) peek() qnode {
	if q.len == 0 {
		return qnode{}
	}
	return q.nodes[1]
}

func (q *queue) pop() qnode {
	if q.len == 0 {
		return qnode{}
	}
	n := q.nodes[1]
	q.nodes[1] = q.nodes[q.len]
	q.len--
	var j, k int
	i := 1
	for i != q.len+1 {
		k = q.len + 1
		j = 2 * i
		if j <= q.len && q.nodes[j].unix < q.nodes[k].unix {
			k = j
		}
		if j+1 <= q.len && q.nodes[j+1].unix < q.nodes[k].unix {
			k = j + 1
		}
		q.nodes[i] = q.nodes[k]
		i = k
	}
	return n
}
