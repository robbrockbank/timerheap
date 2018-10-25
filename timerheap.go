package timerheap

import (
	"container/heap"
	"sync"
	"time"
)

type TimerHeap interface {
	PushEvent(popAfter time.Duration, value interface{})
	TimedEvent() <-chan interface{}
	Terminate()
}

func New() TimerHeap {
	t := &timerHeap{
		wakeup:  make(chan struct{}, 1),
		exit:    make(chan struct{}, 0),
		results: make(chan interface{}, 0),
	}
	go t.run()
	return t
}

type timerHeap struct {
	// Lock to protect access to the heap structure.
	lock      sync.Mutex
	valueHeap timedItemHeap
	// wakeup channel is used to wakeup the event goroutine when a new item that is potentially
	// earlier than the existing one has been added. It is of capacity 1 because we only need
	// a single backed-up wakeup call.
	wakeup chan struct{}
	// exit is used to terminate the event goroutine immediately.
	exit chan struct{}
	// results channel, events are added to this channel when their associated timer pops.
	results chan interface{}
}

func (t *timerHeap) PushEvent(popAfter time.Duration, value interface{}) {
	t.lock.Lock()
	defer t.lock.Unlock()

	ti := timedItem{
		expire: time.Now().Add(popAfter),
		value:  value,
	}
	if next := t.valueHeap.peek(); next == nil || ti.expire.Before(next.expire) {
		// This new item is either the first to be added, or is before the first one in the
		// heap. Send a wakeup to either trigger a timer, or to allow us to recheck if the
		// current timer is still the correct one to be waiting on.
		select {
		case t.wakeup <- struct{}{}:
			// Wakeup sent.
		default:
			// Wakeup already pending.
		}
	}
	heap.Push(&t.valueHeap, ti)
}

func (t *timerHeap) TimedEvent() <-chan interface{} {
	return t.results
}

func (t *timerHeap) Terminate() {
	t.exit <- struct{}{}
	close(t.wakeup)
	close(t.exit)
	close(t.results)
}

func (t *timerHeap) run() {
waitforitem:
	for {
		var ti interface{}
		t.lock.Lock()
		if t.valueHeap.Len() > 0 {
			ti = heap.Pop(&t.valueHeap)
		}
		t.lock.Unlock()

		if ti == nil {
			select {
			case <-t.exit:
				return
			case <-t.wakeup:
				// Woken up, must have an item now.
				continue waitforitem
			}
		}

		tiv := ti.(timedItem)
		wait := tiv.expire.Sub(time.Now())

		if wait <= 0 {
			select {
			case t.results <- tiv.value:
				continue waitforitem
			case <-t.exit:
				return
			}
		}
		tm := time.NewTimer(wait)

	waitfortimer:
		for {
			select {
			case <-t.exit:
				return
			case <-t.wakeup:
				// Woken up, must have an item that potentially has a expire time less than ours.
				t.lock.Lock()
				if next := t.valueHeap.peek(); next != nil && next.expire.Before(tiv.expire) {
					// The next entry on the heap is before the one we were waiting on. Add it
					// back to the heap and reloop to pull the next item.
					heap.Push(&t.valueHeap, tiv)
					t.lock.Unlock()
					continue waitforitem
				}
				t.lock.Unlock()
				continue waitfortimer
			case <-tm.C:
				select {
				case t.results <- tiv.value:
					continue waitforitem
				case <-t.exit:
					return
				}
			}
		}
	}
}

// An timedItemHeap is a min-heap of timedItems, priority is based on the time.
type timedItem struct {
	expire time.Time
	value  interface{}
}
type timedItemHeap []timedItem

// timeItemHeap implements heap.Interface
func (h timedItemHeap) Len() int           { return len(h) }
func (h timedItemHeap) Less(i, j int) bool { return h[i].expire.Before(h[j].expire) }
func (h timedItemHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *timedItemHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(timedItem))
}

func (h *timedItemHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// peek is used to look at the first entry that would be popped off the heap.
func (h *timedItemHeap) peek() *timedItem {
	c := *h
	n := len(c)
	if n == 0 {
		return nil
	}
	return &c[n-1]
}
