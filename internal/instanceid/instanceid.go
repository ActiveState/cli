package instanceid

import (
	"sync"

	"github.com/google/uuid"
)

var (
	id string
	mu sync.Mutex
)

func ID() string {
	mu.Lock()
	defer mu.Unlock()

	if id == "" {
		id = uuid.New().String()
	}
	return id
}
