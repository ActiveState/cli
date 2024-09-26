package workerpool

import (
	"errors"
	"fmt"
	"testing"
)

func TestError(t *testing.T) {
	errToThrow := fmt.Errorf("error")
	wp := New(1)
	wp.Submit(func() error {
		return nil
	})
	wp.Submit(func() error {
		return errToThrow
	})
	err := wp.Wait()
	if !errors.Is(err, errToThrow) {
		t.Errorf("expected error to be %v, got %v", errToThrow, err)
	}
}
