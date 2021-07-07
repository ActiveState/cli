package singlethread

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Singlethread(t *testing.T) {
	tt := New()
	defer tt.Close()
	y := 0
	z := 0

	wg := &sync.WaitGroup{}
	for x := 0; x < 1000; x++ {
		wg.Add(1)
		go tt.Run(func() error {
			defer wg.Done()
			y = y + 1
			return nil
		})
		go func() { z = z + 1 }()
	}
	wg.Wait()
	assert.Equal(t, 1000, y)
	assert.NotEqual(t, 1000, z)
}
