package retries

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"testing"
	"time"
)

type clockMock struct {
	time      time.Time
	durs      []time.Duration
	numSleeps int
}

func (c *clockMock) Sleep(d time.Duration) {
	c.durs = append(c.durs, d)
	c.numSleeps++
	c.time = c.time.Add(d)
}

func (c *clockMock) Now() time.Time {
	return c.time
}

func TestDefault(t *testing.T) {
	t.Run("retry-func", func(t *testing.T) {
		check := false
		r := New(func() error {
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
		c := &clockMock{}

		e := errors.New("my error")

		calls := 0
		r := New(func() error {
			calls++
			return e
		}, WithClock(c), WithRetries(5))

		err := r.Try()

		if !errors.Is(err, e) {
			t.Fatalf("invalid error returned: %s", err)
		}
		if calls != 5 {
			t.Fatal("invalid number of calls")
		}
		if c.numSleeps != 4 {
			t.Fatal("invalid number of sleeps")
		}
		if !reflect.DeepEqual(c.durs, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}) {
			t.Fatalf("invalid sleep durations: %#v", c.durs)
		}
	})

	t.Run("eventual-success", func(t *testing.T) {
		c := &clockMock{}

		e := errors.New("my error")

		calls := 0
		r := NewFull(func(i int, _ time.Time) error {
			calls++
			if i < 4 {
				return e
			}
			return nil
		}, WithClock(c), WithRetries(5))

		err := r.Try()

		if err != nil {
			t.Fatal("invalid error returned")
		}
		if calls != 5 {
			t.Fatal("invalid number of calls")
		}
		if c.numSleeps != 4 {
			t.Fatal("invalid number of sleeps")
		}
		if !reflect.DeepEqual(c.durs, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}) {
			t.Fatalf("invalid sleep durations: %#v", c.durs)
		}
	})
}

func TestWhitelist(t *testing.T) {
	t.Run("simple-whitelist", func(t *testing.T) {
		c := &clockMock{}

		e := errors.New("my error")

		calls := 0
		r := New(func() error {
			calls++
			return e
		}, WithClock(c), WithRetries(5), WithWhitelist(e))

		err := r.Try()

		if !errors.Is(err, e) {
			t.Fatalf("invalid error returned: %s", err)
		}
		if calls != 5 {
			t.Fatal("invalid number of calls")
		}
		if c.numSleeps != 4 {
			t.Fatal("invalid number of sleeps")
		}
		if !reflect.DeepEqual(c.durs, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}) {
			t.Fatalf("invalid sleep durations: %#v", c.durs)
		}
	})

	t.Run("whitelist-redefined-errors", func(t *testing.T) {
		c := &clockMock{}
		calls := 0

		r := New(func() error {
			calls++
			return errors.New("my error")
		}, WithClock(c), WithRetries(5), WithWhitelist(errors.New("my error")))

		err := r.Try()

		if err.Error() != "my error" {
			t.Fatal("invalid error returned")
		}
		if calls != 5 {
			t.Fatalf("invalid number of calls: %d", calls)
		}
		if c.numSleeps != 4 {
			t.Fatal("invalid number of sleeps")
		}
		if !reflect.DeepEqual(c.durs, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}) {
			t.Fatalf("invalid sleep durations: %#v", c.durs)
		}
	})

	t.Run("no-match", func(t *testing.T) {
		c := &clockMock{}

		e := errors.New("my error")

		calls := 0
		r := New(func() error {
			calls++
			return e
		}, WithClock(c), WithRetries(5), WithWhitelist())

		err := r.Try()

		if !errors.Is(err, e) {
			t.Fatal("invalid error returned")
		}
		if calls != 1 {
			t.Fatal("invalid number of calls")
		}
		if c.numSleeps != 0 {
			t.Fatal("invalid number of sleeps")
		}
	})

	t.Run("wrapped-err", func(t *testing.T) {
		c := &clockMock{}

		e := errors.New("my error")

		calls := 0
		r := New(func() error {
			calls++
			return fmt.Errorf("error %w", e)
		}, WithClock(c), WithRetries(5), WithWhitelist(e))

		err := r.Try()

		if !errors.Is(err, e) {
			t.Fatal("invalid error returned")
		}
		if calls != 5 {
			t.Fatal("invalid number of calls")
		}
		if c.numSleeps != 4 {
			t.Fatal("invalid number of sleeps")
		}
		if !reflect.DeepEqual(c.durs, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}) {
			t.Fatalf("invalid sleep durations: %#v", c.durs)
		}
	})

	t.Run("substring error", func(t *testing.T) {
		c := &clockMock{}

		e := errors.New("really long error")

		calls := 0
		r := New(func() error {
			calls++
			return errors.New("my really long error")
		}, WithClock(c), WithRetries(5), WithWhitelist(e))

		err := r.Try()

		if err.Error() != "my really long error" {
			t.Fatalf("invalid error returned: %s", err)
		}
		if calls != 5 {
			t.Fatal("invalid number of calls")
		}
		if c.numSleeps != 4 {
			t.Fatal("invalid number of sleeps")
		}
		if !reflect.DeepEqual(c.durs, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}) {
			t.Fatalf("invalid sleep durations: %#v", c.durs)
		}
	})
}

func ExampleSimple() {
	myFunc := func() error {
		return errors.New("error")
	}

	retrier := New(myFunc)
	if err := retrier.Try(); err != nil {
		log.Println(err)
	}
}

func ExampleFullAPI() {
	myErr := errors.New("error")

	myFunc := func() error {
		return myErr
	}

	retrier := New(
		myFunc,
		WithRetries(10),
		WithWhitelist(myErr),
		WithExpBackoff(2),
	)
	if err := retrier.Try(); err != nil {
		log.Println(err)
	}
}
