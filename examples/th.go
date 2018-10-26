package main

import (
	"fmt"
	"math/rand"
	"time"
	"github.com/robbrockbank/timerheap"
)

type stuff struct {
	idx      int
	expected time.Time
}

const (
	highNumEntries = 10000
)

func main() {
	th := timerheap.New()

	// Simple example use of timerheap
	fmt.Print("Simple 4 events reordered correctly\n\n")

	th.PushEvent(500 * time.Millisecond, stuff{
		2,
		time.Now().Add(500 * time.Millisecond),
	})
	th.PushEvent(1200 * time.Millisecond, stuff{
		4,
		time.Now().Add(1200 * time.Millisecond),
	})
	th.PushEvent(800 * time.Millisecond, stuff{
		3,
		time.Now().Add(800 * time.Millisecond),
	})
	th.PushEvent(50 * time.Millisecond, stuff{
		1,
		time.Now().Add(50 * time.Millisecond),
	})

	for i := 0; i < 4; i++ {
		s := <- th.TimedEvent()
		now := time.Now()
		fmt.Printf("Timer Event:   %d\n", s.(stuff).idx)
		fmt.Printf("  Expected at: %s\n", s.(stuff).expected)
		fmt.Printf("  Popped at:   %s\n", now)
		fmt.Printf("  Delta:       %s\n", now.Sub(s.(stuff).expected))
	}

	// Simple benchmark tests to ensure large number of events doesn't cause slowdowns in
	// timer pops.
	fmt.Printf("\n\nTesting %d events are reordered correctly and determining min/max time\n\n", highNumEntries)

	stuffs := make([]stuff, highNumEntries, highNumEntries)
	for i := 0; i < len(stuffs); i++ {
		stuffs[i] = stuff{idx: i}
	}
	rand.Shuffle(len(stuffs), func(i, j int) {stuffs[i], stuffs[j] = stuffs[j], stuffs[i]})

	now := time.Now()
	for i := 0; i < len(stuffs); i++ {
		expire := now.Add((2 * time.Second) + (time.Duration(stuffs[i].idx) * 10 * time.Millisecond))
		stuffs[i].expected = expire
		th.PushEvent(stuffs[i].expected.Sub(time.Now()), stuffs[i])
	}

	fmt.Printf("Receiving %d events\n", highNumEntries)
	deltaMin := time.Hour
	deltaMax := -time.Hour
	for i := 0; i < len(stuffs); i++ {
		s := <- th.TimedEvent()
		now := time.Now()
		delta := now.Sub(s.(stuff).expected)
		if delta < deltaMin {
			deltaMin = delta
		}
		if delta > deltaMax {
			deltaMax = delta
		}
		if i != s.(stuff).idx {
			fmt.Printf("Index out of order: expected=%d; got=%d\n", i, s.(stuff).idx)
		}
		if i % 100 == 0 {
			fmt.Print(".")
		}
	}
	fmt.Printf("\nReceived all %d events\n", highNumEntries)
	fmt.Printf("  Min delta: %s\n", deltaMin)
	fmt.Printf("  Max delta: %s\n", deltaMax)

	th.Terminate()
}


