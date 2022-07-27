package instanceid

import (
	"sync"

	"github.com/google/uuid"
)

func Make() string {
	return uuid.New().String()
}

var (
	id string
	mu sync.Mutex
)

func ID() string {
	mu.Lock()
	defer mu.Unlock()

	if id == "" {
		id = Make()
	}
	return id
}
