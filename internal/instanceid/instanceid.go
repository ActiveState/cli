package instanceid

import "github.com/google/uuid"

var id string

func ID() string {
	if id == "" {
		id = uuid.New().String()
	}
	return id
}
