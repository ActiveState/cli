package retryfn

import (
	"errors"
	"reflect"
	"testing"
)

var errStoreMax = errors.New("max stored")

type store struct {
	max int
	ns  []int
}

func (s *store) bump(n int) error {
	if len(s.ns) >= s.max {
		return errStoreMax
	}
	s.ns = append(s.ns, n)
	return nil
}

func TestRetryFn(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		out := []int{1, 2, 3, 4, 5, 6}
		var ns []int

		var i int
		fn := func() error {
			i++
			ns = append(ns, i)
			return nil
		}

		retryFn := New(len(out), fn)

		err := retryFn.Run()
		if err != nil {
			t.Fatalf("run error: got %v, want nil", err)
		}

		got := retryFn.Calls()
		if got != i {
			t.Fatalf("call count: got %v, want %v", got, i)
		}

		if !reflect.DeepEqual(ns, out) {
			t.Errorf("equal slices: got %v, want %v", ns, out)
		}
	})

	t.Run("force error via max", func(t *testing.T) {
		var ns []int

		s := store{max: 6}
		extraIterations := 3

		var i int
		fn := func() error {
			i++
			if err := s.bump(i); err != nil {
				return err
			}
			ns = append(ns, i)
			return nil
		}

		retryFn := New(s.max+extraIterations, fn)

		err := retryFn.Run()
		if !errors.Is(err, errStoreMax) {
			t.Fatalf("run error: got %v, want ErrStoreMax", err)
		}

		got := retryFn.Calls()
		if got != i {
			t.Fatalf("call count: got %v, want %v", got, i)
		}

		if !reflect.DeepEqual(s.ns, ns) {
			t.Errorf("equal slices: got %v, want %v", s.ns, ns)
		}
	})
}
