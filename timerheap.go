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
		// This new item is either the first to be added, or expires before the first one in the
		// heap. Send a wakeup to trigger the timer thread to recheck.
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

		// Determine how long we need to wait for this item to expire.
		tiv := ti.(timedItem)
		wait := tiv.expire.Sub(time.Now())

		// If this item has expired, then send immediately rather than going to the extremes
		// of creating a timer with a negative duration.
		if wait <= 0 {
			select {
			case t.results <- tiv.value:
				continue waitforitem
			case <-t.exit:
				return
			}
		}

		// The event expires in the future, so use a channel based timer to wait for the event - this
		// makes it easy to cancel if the timerheap is terminated, or a new event has been added which
		// may have a closer expiration time.
		tm := time.NewTimer(wait)

	waitfortimer:
		for {
			select {
			case <-t.exit:
				tm.Stop()
				return
			case <-t.wakeup:
				// Woken up, must have an item that potentially has a expire time less than ours.
				t.lock.Lock()
				if next := t.valueHeap.peek(); next != nil && next.expire.Before(tiv.expire) {
					// The next entry on the heap is before the one we were waiting on. Add it
					// back to the heap, cancel it's timer and reloop to pull the next item
					// which will have a closer expiration.
					heap.Push(&t.valueHeap, tiv)
					t.lock.Unlock()
					tm.Stop()
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

// As per heap.Interface, Push appends an item after the last index.
func (h *timedItemHeap) Push(x interface{}) {
	*h = append(*h, x.(timedItem))
}

// As per heap.Interface, Pop removes the item at index 0.
func (h *timedItemHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// peek is used to look at the first entry that would be popped off the heap (which is
// the first element in the slice).
//
// This uses knowledge (as defined by the heap.Pop(), that Pop is equivalent to Remove(h, 0).
func (h *timedItemHeap) peek() *timedItem {
	if h.Len() == 0 {
		return nil
	}
	c := *h
	return &c[0]
}
