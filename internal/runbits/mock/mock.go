package mock

import (
	"time"
)

// Incrementer implements a simple counter.  This can be used to test functions that are expected to report its progress incrementally
type Incrementer struct {
	Count int
}

// NewMockIncrementer fakes an integral progresser
func NewMockIncrementer() *Incrementer {
	return &Incrementer{Count: 0}
}

// Increment increments the progress count by one
func (mi *Incrementer) Increment(_ ...time.Duration) {
	mi.Count++
}
