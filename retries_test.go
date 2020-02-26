package retries

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

type sleeperMock struct {
	durs      []time.Duration
	numSleeps int
}

func (s *sleeperMock) Sleep(d time.Duration) {
	s.durs = append(s.durs, d)
	s.numSleeps++
}

func TestDefault(t *testing.T) {
	t.Run("retry-func", func(t *testing.T) {
		check := false
		r := New(func(i int) error {
			check = true
			return nil
		})

		if err := r.Try(); err != nil {
			t.Fatal(err)
		}

		if !check {
			t.Fatal("retry func not called")
		}
	})

	t.Run("retries-run", func(t *testing.T) {
		s := &sleeperMock{}

		e := errors.New("my error")

		calls := 0
		r := New(func(i int) error {
			calls++
			return e
		}, WithSleeper(s), WithRetries(5))

		err := r.Try()

		if !errors.Is(err, e) {
			t.Fatal("invalid error returned")
		}
		if calls != 5 {
			t.Fatal("invalid number of calls")
		}
		if s.numSleeps != 4 {
			t.Fatal("invalid number of sleeps")
		}
		if !reflect.DeepEqual(s.durs, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}) {
			t.Fatalf("invalid sleep durations: %#v", s.durs)
		}
	})

	t.Run("eventual-success", func(t *testing.T) {
		s := &sleeperMock{}

		e := errors.New("my error")

		calls := 0
		r := New(func(i int) error {
			calls++
			if i < 4 {
				return e
			}
			return nil
		}, WithSleeper(s), WithRetries(5))

		err := r.Try()

		if err != nil {
			t.Fatal("invalid error returned")
		}
		if calls != 5 {
			t.Fatal("invalid number of calls")
		}
		if s.numSleeps != 4 {
			t.Fatal("invalid number of sleeps")
		}
		if !reflect.DeepEqual(s.durs, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}) {
			t.Fatalf("invalid sleep durations: %#v", s.durs)
		}
	})
}

func TestWhitelist(t *testing.T) {
	t.Run("simple-whitelist", func(t *testing.T) {
		s := &sleeperMock{}

		e := errors.New("my error")

		calls := 0
		r := New(func(i int) error {
			calls++
			return e
		}, WithSleeper(s), WithRetries(5), WithWhitelist(e))

		err := r.Try()

		if !errors.Is(err, e) {
			t.Fatal("invalid error returned")
		}
		if calls != 5 {
			t.Fatal("invalid number of calls")
		}
		if s.numSleeps != 4 {
			t.Fatal("invalid number of sleeps")
		}
		if !reflect.DeepEqual(s.durs, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}) {
			t.Fatalf("invalid sleep durations: %#v", s.durs)
		}
	})

	t.Run("whitelist-redefined-errors", func(t *testing.T) {
		s := &sleeperMock{}
		calls := 0

		r := New(func(i int) error {
			calls++
			return errors.New("my error")
		}, WithSleeper(s), WithRetries(5), WithWhitelist(errors.New("my error")))

		err := r.Try()

		if err.Error() != "my error" {
			t.Fatal("invalid error returned")
		}
		if calls != 5 {
			t.Fatalf("invalid number of calls: %d", calls)
		}
		if s.numSleeps != 4 {
			t.Fatal("invalid number of sleeps")
		}
		if !reflect.DeepEqual(s.durs, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}) {
			t.Fatalf("invalid sleep durations: %#v", s.durs)
		}
	})

	t.Run("no-match", func(t *testing.T) {
		s := &sleeperMock{}

		e := errors.New("my error")

		calls := 0
		r := New(func(i int) error {
			calls++
			return e
		}, WithSleeper(s), WithRetries(5), WithWhitelist())

		err := r.Try()

		if !errors.Is(err, e) {
			t.Fatal("invalid error returned")
		}
		if calls != 1 {
			t.Fatal("invalid number of calls")
		}
		if s.numSleeps != 0 {
			t.Fatal("invalid number of sleeps")
		}
	})

	t.Run("wrapped-err", func(t *testing.T) {
		s := &sleeperMock{}

		e := errors.New("my error")

		calls := 0
		r := New(func(i int) error {
			calls++
			return fmt.Errorf("error %w", e)
		}, WithSleeper(s), WithRetries(5), WithWhitelist(e))

		err := r.Try()

		if !errors.Is(err, e) {
			t.Fatal("invalid error returned")
		}
		if calls != 5 {
			t.Fatal("invalid number of calls")
		}
		if s.numSleeps != 4 {
			t.Fatal("invalid number of sleeps")
		}
		if !reflect.DeepEqual(s.durs, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}) {
			t.Fatalf("invalid sleep durations: %#v", s.durs)
		}
	})
}
