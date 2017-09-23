package queue

import (
	"errors"
	"sync"
)

type Compare func(interface{}, interface{}) bool

// BlockingQueue is a FIFO queue where Pop() operation is blocking if no items exists
type BlockingQueue struct {
	closed bool
	lock   sync.Mutex
	queue  []interface{}

	notifyLock sync.Mutex
	monitor    *sync.Cond
}

// New instance of FIFO queue
func New() *BlockingQueue {
	bq := &BlockingQueue{}
	bq.monitor = sync.NewCond(&bq.notifyLock)
	return bq
}

// Put any value to queue back. Returns false if queue closed
func (bq *BlockingQueue) Put(value interface{}) bool {
	if bq.closed {
		return false
	}
	bq.lock.Lock()
	if bq.closed {
		return false
	}
	bq.queue = append(bq.queue, value)
	bq.lock.Unlock()

	bq.notifyLock.Lock()
	bq.monitor.Signal()
	bq.notifyLock.Unlock()
	return true
}

func (bq *BlockingQueue) Remove(value interface{}, equal Compare) bool {
	for i, src := range bq.queue {
		if equal(src, value) == true {
			if bq.closed {
				return false
			}
			bq.lock.Lock()
			if bq.closed {
				return false
			}
			bq.queue = append(bq.queue[:i], bq.queue[i+1:]...)
			bq.lock.Unlock()
			bq.notifyLock.Lock()
			bq.monitor.Signal()
			bq.notifyLock.Unlock()
			return true
		}
	}
	return false
}

func (bq *BlockingQueue) Get(value interface{}, equal Compare) interface{} {
	for _, src := range bq.queue {
		if equal(src, value) == true {
			if bq.closed {
				return nil
			}
			return src
		}
	}
	return nil
}

func (bq *BlockingQueue) Iterator() <-chan interface{} {
	channel := make(chan interface{})
	go func() {
		for _, val := range bq.queue {
			channel <- val
		}
		close(channel)
	}()
	return channel
}

// Pop front value from queue. Returns nil and false if queue closed
func (bq *BlockingQueue) Pop() interface{} {
	var output interface{}
	if bq.closed {
		return nil
	}
	bq.lock.Lock()
	if bq.closed {
		return nil
	}
	output, bq.queue = bq.queue[len(bq.queue)-1], bq.queue[:len(bq.queue)-1]
	bq.lock.Unlock()
	bq.notifyLock.Lock()
	bq.monitor.Signal()
	bq.notifyLock.Unlock()
	return output
}

// Size of queue. Performance is O(1)
func (bq *BlockingQueue) Size() int {
	bq.lock.Lock()
	defer bq.lock.Unlock()
	return len(bq.queue)
}

// Closed flag
func (bq *BlockingQueue) Closed() bool {
	bq.lock.Lock()
	defer bq.lock.Unlock()
	return bq.closed
}

// Close queue and explicitly remove each item from queue.
// Also notifies all reader (they will return nil and false)
// Returns error if queue already closed
func (bq *BlockingQueue) Close() error {
	if bq.closed {
		return errors.New("Already closed")
	}
	bq.closed = true
	bq.lock.Lock()
	//Clear
	bq.queue = bq.queue[:0]
	bq.lock.Unlock()
	bq.monitor.Broadcast()
	return nil
}

func (bq *BlockingQueue) Empty() error {
	if bq.closed {
		return nil
	}
	bq.lock.Lock()
	//Clear
	bq.queue = bq.queue[:0]
	bq.lock.Unlock()
	bq.monitor.Broadcast()
	return nil
}
