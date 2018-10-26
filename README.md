# timerheap

Timer heap is an implementation of a timed event queue using a heap to order the
events. Events with different timer pop times may be added.

Usage:

```go
package main

import (
	"fmt"
	"time"
	"github.com/robbrockbank/timerheap"
)

type stuff struct {
	idx      int
	expected time.Time
}

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
	
    th.Terminate()
}
```

Output:
```
Simple 4 events reordered correctly

Timer Event:   1
  Expected at: 2018-10-25 19:35:24.667569044 -0700 PDT m=+0.050508494
  Popped at:   2018-10-25 19:35:24.668635396 -0700 PDT m=+0.051576731
  Delta:       1.068237ms
Timer Event:   2
  Expected at: 2018-10-25 19:35:25.117540129 -0700 PDT m=+0.500479578
  Popped at:   2018-10-25 19:35:25.118625312 -0700 PDT m=+0.501583254
  Delta:       1.103676ms
Timer Event:   3
  Expected at: 2018-10-25 19:35:25.417564684 -0700 PDT m=+0.800504133
  Popped at:   2018-10-25 19:35:25.418583967 -0700 PDT m=+0.801552979
  Delta:       1.048846ms
Timer Event:   4
  Expected at: 2018-10-25 19:35:25.817563862 -0700 PDT m=+1.200503311
  Popped at:   2018-10-25 19:35:25.817679736 -0700 PDT m=+1.200663478
  Delta:       160.167µs

```