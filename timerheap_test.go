package timerheap_test

import (
	"time"

	"math/rand"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/robbrockbank/timerheap"
)

const (
	initialPause = 500 * time.Millisecond
	interval = 500 * time.Millisecond
	accuracy = 100 * time.Millisecond
)

type testdata struct {
	index int
	pop   time.Time
}

var _ = Describe("timer heap tests", func() {

	var th timerheap.TimerHeap
	var td []testdata
	var now time.Time

	Context("timer heap test event ordering", func() {
		BeforeEach(func() {
			th = timerheap.New()
		})

		AfterEach(func() {
			th.Terminate()
		})

		It("gets future events add in random order in the correct order", func() {
			var value interface{}

			By("adding a past event that will end up blocking the results channel")
			th.PushEvent(-time.Hour, testdata{
				index: 9000,
			})

			By("Pausing to ensure this first past event is waiting to be sent")
			time.Sleep(initialPause)

			By("creating a set of future events")
			td = make([]testdata, 50, 50)
			now = time.Now()
			for i := 0; i < len(td); i++ {
				td[i] = testdata{
					index: i,
					pop:   now.Add(interval * time.Duration(i+1)),
				}
			}

			By("Adding them to the timer in a random order")
			rand.Shuffle(len(td), func(i, j int) { td[i], td[j] = td[j], td[i] })
			for i := 0; i < len(td); i++ {
				th.PushEvent(td[i].pop.Sub(time.Now()), td[i])
			}

			By("Waiting for the first event")
			Eventually(th.TimedEvent(), "1s", "10ms").Should(Receive(&value))
			Expect(value).NotTo(BeNil())
			t, ok := value.(testdata)
			Expect(ok).To(BeTrue())
			Expect(t.index).To(Equal(9000))

			By("Waiting for the events and checking the order is correct")
			for i := 0; i < len(td); i++ {
				Eventually(th.TimedEvent(), "1s", "10ms").Should(Receive(&value))
				Expect(value).NotTo(BeNil())
				t, ok := value.(testdata)
				Expect(ok).To(BeTrue())
				Expect(t.index).To(Equal(i))
				delta := time.Now().Sub(t.pop)
				Expect(delta).To(BeNumerically(">", -accuracy))
				Expect(delta).To(BeNumerically("<", accuracy))
			}
		})

		It("gets past events added in random order in the correct order", func() {
			var value interface{}

			By("adding a past event that will end up blocking the results channel")
			th.PushEvent(-time.Hour, testdata{
				index: 9000,
			})

			By("Pausing to ensure this first past event is waiting to be sent")
			time.Sleep(initialPause)

			By("creating a set of past events")
			td = make([]testdata, 50, 50)
			now = time.Now()
			for i := 0; i < len(td); i++ {
				td[i] = testdata{
					index: i,
					pop:   now.Add(-interval * time.Duration(i+1)),
				}
			}

			By("Adding them to the timer in a random order")
			rand.Shuffle(len(td), func(i, j int) { td[i], td[j] = td[j], td[i] })
			for i := 0; i < len(td); i++ {
				th.PushEvent(td[i].pop.Sub(time.Now()), td[i])
			}

			By("Waiting for the first event")
			Eventually(th.TimedEvent(), "1s", "10ms").Should(Receive(&value))
			Expect(value).NotTo(BeNil())
			t, ok := value.(testdata)
			Expect(ok).To(BeTrue())
			Expect(t.index).To(Equal(9000))

			By("Waiting for the events and checking the order is correct")
			// index values should be in reverse order.
			for i := len(td) - 1; i >= 0; i-- {
				Eventually(th.TimedEvent(), "1s", "10ms").Should(Receive(&value))
				Expect(value).NotTo(BeNil())
				t, ok := value.(testdata)
				Expect(ok).To(BeTrue())
				Expect(t.index).To(Equal(i))
			}
		})

		It("can requeue a pending timer if an earlier event is added (earlier than next in queue)", func() {
			var value interface{}

			By("adding two future events")
			th.PushEvent(time.Second, testdata{
				index: 1, pop: time.Now(),
			})
			th.PushEvent(time.Second, testdata{
				index: 1, pop: time.Now(),
			})

			By("Pausing for the timer to start but not to pop")
			time.Sleep(100 * time.Millisecond)

			By("adding an event that will expire sooner")
			th.PushEvent(1 * time.Millisecond, testdata{
				index: 2000, pop: time.Now(),
			})

			By("Waiting checking the order of the events")
			Eventually(th.TimedEvent(), "1s", "10ms").Should(Receive(&value))
			Expect(value).NotTo(BeNil())
			t, ok := value.(testdata)
			Expect(ok).To(BeTrue())
			Expect(t.index).To(Equal(2000))

			Eventually(th.TimedEvent(), "3s", "10ms").Should(Receive(&value))
			Expect(value).NotTo(BeNil())
			t, ok = value.(testdata)
			Expect(ok).To(BeTrue())
			Expect(t.index).To(Equal(1))

			Eventually(th.TimedEvent(), "3s", "10ms").Should(Receive(&value))
			Expect(value).NotTo(BeNil())
			t, ok = value.(testdata)
			Expect(ok).To(BeTrue())
			Expect(t.index).To(Equal(1))
		})

		It("can requeue a pending timer if an earlier event is added (no next in queue)", func() {
			var value interface{}

			By("adding one future events")
			th.PushEvent(time.Second, testdata{
				index: 1, pop: time.Now(),
			})

			By("Pausing for the timer to start but not to pop")
			time.Sleep(500 * time.Millisecond)

			By("adding an event that will expire sooner")
			th.PushEvent(0, testdata{
				index: 2000, pop: time.Now(),
			})

			By("Waiting checking the order of the events")
			Eventually(th.TimedEvent(), "1s", "10ms").Should(Receive(&value))
			Expect(value).NotTo(BeNil())
			t, ok := value.(testdata)
			Expect(ok).To(BeTrue())
			Expect(t.index).To(Equal(2000))

			Eventually(th.TimedEvent(), "1s", "10ms").Should(Receive(&value))
			Expect(value).NotTo(BeNil())
			t, ok = value.(testdata)
			Expect(ok).To(BeTrue())
			Expect(t.index).To(Equal(1))
		})
	})

	Context("termination processing", func() {
		BeforeEach(func() {
			th = timerheap.New()
		})

		It("can terminate before adding any events", func() {
			By("Terminating the timer")
			th.Terminate()
		})

		It("can terminate before receiving a past-time events", func() {
			By("Adding an event at a previous time")
			th.PushEvent(0, testdata{
				index: 0, pop: time.Now().Add(-time.Hour),
			})

			By("Terminating the timer")
			th.Terminate()
		})

		It("can terminate before receiving anything", func() {
			By("adding a set of immediate events")
			for i := 0; i < 50; i++ {
				th.PushEvent(0, testdata{
					index: i, pop: time.Now(),
				})
			}

			By("Terminating the timer")
			th.Terminate()
		})

		It("can terminate before receiving a future event that hasn't popped yet", func() {
			By("adding a set of immediate events")
			th.PushEvent(time.Second, testdata{
				index: 1, pop: time.Now(),
			})

			By("Pausing for the timer to start, but not to pop")
			time.Sleep(500 * time.Millisecond)

			By("Terminating the timer")
			th.Terminate()
		})

		It("can terminate before receiving a future event that hasn't popped yet", func() {
			By("adding a set of immediate events")
			th.PushEvent(500 * time.Millisecond, testdata{
				index: 1, pop: time.Now(),
			})

			By("Pausing for the timer to start and to pop")
			time.Sleep(time.Second)

			By("Terminating the timer")
			th.Terminate()
		})

		It("can terminate before receiving everything", func() {
			By("adding a set of immediate events")
			for i := 0; i < 50; i++ {
				th.PushEvent(0, testdata{
					index: i, pop: time.Now(),
				})
			}

			By("Pulling half of the events")
			for i := 0; i < len(td)/2; i++ {
				var value interface{}
				Eventually(th.TimedEvent(), "1s", "10ms").Should(Receive(&value))
			}

			By("Terminating the timer")
			th.Terminate()
		})
	})
})
