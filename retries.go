package retries

import (
	"errors"
	"math"
	"strings"
	"time"
)

const (
	defaultRetries       = 4
	defaultBackoffFactor = 2
)

type (
	clock struct{}

	// Clock is an interface that offers several clock functions.
	Clock interface {
		Now() time.Time
		Sleep(time.Duration)
	}

	// Arg is a parameter to New.
	Arg func(*Retrier) *Retrier

	// Func is a function that can be retried.
	Func func() error

	// FullFunc is a function that can be retried with an extended
	// interface with access to retry metadata.
	FullFunc func(retryNum int, lastRetry time.Time) error

	// Retrier is a type that manages retries.
	Retrier struct {
		f             interface{}
		clock         Clock
		retries       int
		retryCheck    func(error) bool
		sleepStrategy func(int, Clock)
	}
)

// New initializes a new Retrier. By default (with no other Args), the
// Retrier retries all errors with an exponential backoff strategy
// three times. This behavior can be customized with functional
// arguments using the functions in this package.
func New(f Func, args ...Arg) *Retrier {
	r := &Retrier{
		f: f,
	}
	for _, a := range args {
		a(r)
	}

	r.setDefaults()

	return r
}

// NewFull initializes a new Retrier. This behaves similar to New, but
// accepting FullFunc functions, which offer an extended interface.
func NewFull(f FullFunc, args ...Arg) *Retrier {
	r := &Retrier{
		f: f,
	}
	for _, a := range args {
		a(r)
	}

	r.setDefaults()

	return r
}

// Try runs the retry process until the number of retries is
// exhausted.
func (r *Retrier) Try() error {
	var err error
	var lastRetryTime time.Time

	for i := 0; i < r.retries; i++ {
		startTime := r.clock.Now()

		if f, ok := r.f.(Func); ok {
			err = f()
		} else if f, ok := r.f.(FullFunc); ok {
			err = f(i, lastRetryTime)
		} else {
			panic("invalid function interface")
		}

		lastRetryTime = startTime

		if err == nil {
			return nil
		}

		if (i != r.retries-1) && r.retryCheck(err) {
			r.sleepStrategy(i, r.clock)
			continue
		}

		break
	}

	return err
}

func (r *Retrier) setDefaults() {
	if r.clock == nil {
		r.clock = &clock{}
	}
	if r.retries == 0 {
		r.retries = defaultRetries
	}
	if r.retryCheck == nil {
		r.retryCheck = RetryOnAllErrors
	}
	if r.sleepStrategy == nil {
		WithExpBackoff(defaultBackoffFactor)(r)
	}
}

func (c *clock) Sleep(d time.Duration) {
	time.Sleep(d)
}

func (c *clock) Now() time.Time {
	return time.Now()
}

// RetryOnAllErrors is a retry check that retries on all errors.
func RetryOnAllErrors(err error) bool {
	return err != nil
}

// WithRetries sets the number of retries for a Retrier.
func WithRetries(retries int) Arg {
	return func(r *Retrier) *Retrier {
		r.retries = retries

		return r
	}
}

// WithExpBackoff defines an exponential back-off with factor as a
// base. For retry number `i`, this strategy sleeps for `factor ** i`
// seconds.
func WithExpBackoff(factor int) Arg {
	return func(r *Retrier) *Retrier {
		r.sleepStrategy = func(retryNum int, clock Clock) {
			s := math.Pow(float64(factor), float64(retryNum))
			clock.Sleep(time.Second * time.Duration(s))
		}

		return r
	}
}

// WithConstantBackoff defines a back-off strategy with where the
// sleep time is constant between retries.
func WithConstantBackoff(backoff time.Duration) Arg {
	return func(r *Retrier) *Retrier {
		r.sleepStrategy = func(retryNum int, clock Clock) {
			clock.Sleep(backoff)
		}

		return r
	}
}

// WithWhitelist defines a retry condition where the error has to be
// contained within a whitelist of errors. Errors are compared using
// `errors.Is` from the standard library, by comparing error strings
// after unwrapping errors (using stdlib error wrapping and pkg/errors
// error wrapping), and by checking if the whitelisted error is a
// substring of the returned error.
func WithWhitelist(whitelist ...error) Arg {
	return func(r *Retrier) *Retrier {
		r.retryCheck = func(err error) bool {
			for _, e := range whitelist {
				if errors.Is(err, e) {
					return true
				}

				// Stdlib error wrapping
				if err, ok := err.(interface {
					Unwrap() error
				}); ok {
					if err.Unwrap().Error() == e.Error() ||
						strings.Contains(err.Unwrap().Error(), e.Error()) {

						return true
					}
				}

				// pkg/errors error wrapping
				if err, ok := err.(interface {
					Cause() error
				}); ok {
					if err.Cause().Error() == e.Error() ||
						strings.Contains(err.Cause().Error(), e.Error()) {

						return true
					}
				}

				if err.Error() == e.Error() ||
					strings.Contains(err.Error(), e.Error()) {

					return true
				}
			}

			return false
		}

		return r
	}
}

// WithClock sets a custom clock type for the Retrier. Use this to
// mock out the time calls a Retrier makes.
func WithClock(c Clock) Arg {
	return func(r *Retrier) *Retrier {
		r.clock = c

		return r
	}
}

// WithRetryCheck allows the caller to customize the function that
// determines whether an error should be retried.
func WithRetryCheck(chk func(error) bool) Arg {
	return func(r *Retrier) *Retrier {
		r.retryCheck = chk

		return r
	}
}

// WithSleepStrategy sets a custom sleeping strategy. This function
// runs after we've determined that a retry should occur. The
// arguments to the strategy are the retry number and the clock.
func WithSleepStrategy(strategy func(retryNum int, clock Clock)) Arg {
	return func(r *Retrier) *Retrier {
		r.sleepStrategy = strategy

		return r
	}
}
