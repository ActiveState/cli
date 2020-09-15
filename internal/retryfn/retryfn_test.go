package retryfn

import (
	"errors"
	"reflect"
	"testing"
)

var (
	errStoreMax    = errors.New("max stored")
	errNoNegatives = errors.New("will not handle negative integers")
)

type store struct {
	max int
	ns  []int
}

func (s *store) bump(n int) error {
	if n < 0 {
		return &ControlError{
			Cause: errNoNegatives,
			Type:  Halt,
		}
	}

	if len(s.ns) >= s.max {
		return errStoreMax
	}

	s.ns = append(s.ns, n)
	return nil
}

func TestRetryFn(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		out := []int{1, 2, 3, 4, 5, 6}
		s := store{max: len(out)}

		var i int
		fn := func() error {
			i++
			return s.bump(i)
		}

		retryFn := New(len(out), fn)

		err := retryFn.Run()
		if err != nil {
			t.Fatalf("run error: got %v, want nil", err)
		}

		gotCalls := retryFn.Calls()
		wantCalls := len(out)
		if gotCalls != wantCalls {
			t.Fatalf("call count: got %v, want %v", gotCalls, wantCalls)
		}

		if !reflect.DeepEqual(s.ns, out) {
			t.Errorf("equal slices: got %v, want %v", s.ns, out)
		}
	})

	t.Run("force error via max and continue calling", func(t *testing.T) {
		out := []int{1, 2, 3, 4, 5, 6}

		s := store{max: len(out)}
		tryTotal := s.max + 3

		var i int
		fn := func() error {
			i++
			return s.bump(i)
		}

		retryFn := New(tryTotal, fn)

		err := retryFn.Run()
		if !errors.Is(err, errStoreMax) {
			t.Fatalf("run error: got %v, want errStoreMax", err)
		}

		gotCalls := retryFn.Calls()
		if gotCalls != tryTotal {
			t.Fatalf("call count: got %v, want %v", gotCalls, tryTotal)
		}

		if !reflect.DeepEqual(s.ns, out) {
			t.Errorf("equal slices: got %v, want %v", s.ns, out)
		}
	})

	t.Run("halt immediately", func(t *testing.T) {
		var out []int

		s := store{max: 9001}

		fn := func() error {
			return s.bump(-1)
		}

		retryFn := New(s.max, fn)

		err := retryFn.Run()
		if !errors.Is(err, errNoNegatives) {
			t.Fatalf("run error: got %v, want errNoNegatives", err)
		}

		gotCalls := retryFn.Calls()
		wantCalls := 1
		if gotCalls != wantCalls {
			t.Fatalf("call count: got %v, want %v", gotCalls, wantCalls)
		}

		if !reflect.DeepEqual(s.ns, out) {
			t.Errorf("equal slices: got %v, want %v", s.ns, out)
		}
	})
}
