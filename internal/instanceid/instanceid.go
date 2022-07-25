package instanceid

import (
	"sync"

	"github.com/google/uuid"
)

func Make() string {
	return uuid.New().String()
}

var (
	appID string
	mu    sync.Mutex
)

func AppID() string {
	mu.Lock()
	defer mu.Unlock()

	if appID == "" {
		appID = Make()
	}
	return appID
}
