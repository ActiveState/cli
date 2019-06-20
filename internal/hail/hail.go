package hail

import (
	"os"

	"github.com/ActiveState/cli/internal/failures"
)

// Send sends a hail by saving data to the file located by the file name
// provided.
func Send(file string, data []byte) *failures.Failure {
	return nil
}

// Open opens a channel for hailing. A *os.File is returned when the file
// located by the file name provided is created, updated, or deleted.
func Open(file string) (<-chan *os.File, *failures.Failure) {
	return nil, nil
}
