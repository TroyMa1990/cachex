package cachex

// wencan
// 2017-09-02 10:48

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSentinel_Wait(t *testing.T) {
	sentinel := NewSentinel()

	var sum int

	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			value, _ := sentinel.Wait()

			mu.Lock()
			defer mu.Unlock()
			sum += value.(int)
		}()
	}

	sentinel.Done(1, nil)

	wg.Wait()

	assert.Equal(t, 10, sum)
}
