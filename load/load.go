package load

import (
	"math/rand"
	"runtime"
	"sync"
	"time"
)

// The following functions are artificial load generators
func emptySelect() {
	n := runtime.NumCPU()
	runtime.GOMAXPROCS(n)

	quit := make(chan bool)

	for i := 0; i < n; i++ {
		go func() {
			for {
				select {
				case <-quit:
					return
				default:
				}
			}
		}()
	}
	time.Sleep(20 * time.Second)
	for i := 0; i < n; i++ {
		quit <- true
	}
}

// Adapted from https://jvns.ca/blog/2017/09/24/profiling-go-with-pprof/
func leakyFunction() {
	s := make([]string, 3)
	for i := 0; i < 128; i++ {
		s = append(s, "magical pandas")
		if (i % 32) == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func load() {
	for i := 0; i < (1 << 7); i++ {
		rand.Int63()
	}
}

// GenerateArtificialLoad is a throwaway function that produces some
// lambda load so that profiling has data to sample
//
func GenerateArtificialLoad() {
	var once sync.Once
	once.Do(func() {
		go emptySelect()
	})
	go leakyFunction()
	load()
}
