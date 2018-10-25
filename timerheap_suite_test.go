package timerheap

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestTimerHeap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "timerheap suite")
}
