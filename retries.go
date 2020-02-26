package retries

import (
	"errors"
	"math"
	"time"
)

const (
	defaultRetries       = 3
	defaultBackoffFactor = 2
)

type (
	sleeper struct{}

	// Sleeper is a type that can sleep.
	Sleeper interface {
		Sleep(time.Duration)
	}

	// Arg is a parameter to New.
	Arg func(*Retrier) *Retrier

	// Func is a function that can be retried. Its input is the
	// retry number.
	Func func(retryNum int) error

	// Retrier is a type that manages retries.
	Retrier struct {
		f            Func
		sleeper      Sleeper
		retries      int
		retryCheck   func(error) bool
		retrySleeper func(int, Sleeper)
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

// Try runs the retry process until the number of retries is
// exhausted.
func (r *Retrier) Try() error {
	var err error

	for i := 0; i < r.retries; i++ {
		err = r.f(i)
		if err == nil {
			return nil
		}

		if (i != r.retries-1) && r.retryCheck(err) {
			r.retrySleeper(i, r.sleeper)
			continue
		}

		break
	}

	return err
}

func (r *Retrier) setDefaults() {
	if r.sleeper == nil {
		r.sleeper = &sleeper{}
	}
	if r.retries == 0 {
		r.retries = defaultRetries
	}
	if r.retryCheck == nil {
		r.retryCheck = defaultRetryCheck
	}
	if r.retrySleeper == nil {
		WithExpBackoff(defaultBackoffFactor)(r)
	}
}

func (s *sleeper) Sleep(d time.Duration) {
	time.Sleep(d)
}

func defaultRetryCheck(err error) bool {
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
		r.retrySleeper = func(retryNum int, sleeper Sleeper) {
			s := math.Pow(float64(factor), float64(retryNum))
			sleeper.Sleep(time.Second * time.Duration(s))
		}

		return r
	}
}

// WithConstantBackoff defines a back-off strategy with where the
// sleep time is constant between retries.
func WithConstantBackoff(backoff time.Duration) Arg {
	return func(r *Retrier) *Retrier {
		r.retrySleeper = func(retryNum int, sleeper Sleeper) {
			sleeper.Sleep(backoff)
		}

		return r
	}
}

// WithWhitelist defines a retry condition where the error has to be
// contained within a whitelist of errors. Errors are compared using
// `errors.Is` from the standard library.
func WithWhitelist(whitelist ...error) Arg {
	return func(r *Retrier) *Retrier {
		r.retryCheck = func(err error) bool {
			for _, e := range whitelist {
				if errors.Is(err, e) {
					return true
				}
			}

			return false
		}

		return r
	}
}

// WithBlacklist defines a retry condition where the error has to not
// be contained within a blacklist of errors. Errors are compared
// using `errors.Is` from the standard library.
func WithBlacklist(blacklist ...error) Arg {
	return func(r *Retrier) *Retrier {
		r.retryCheck = func(err error) bool {
			for _, e := range blacklist {
				if errors.Is(err, e) {
					return false
				}
			}

			return true
		}

		return r
	}
}

// WithSleeper sets a custom sleeping type. Use this to mock out the
// sleeping calls a Retrier makes.
func WithSleeper(s Sleeper) Arg {
	return func(r *Retrier) *Retrier {
		r.sleeper = s
		return r
	}
}

// WithRetryCheck determines whether an error should be retried.
func WithRetryCheck(chk func(error) bool) Arg {
	return func(r *Retrier) *Retrier {
		r.retryCheck = chk

		return r
	}
}

// WithSleepStrategy sets a custom sleeping strategy. The arguments to
// the strategy are the retry number and the sleeper.
func WithSleepStrategy(strategy func(retryNum int, sleeper Sleeper)) Arg {
	return func(r *Retrier) *Retrier {
		r.retrySleeper = strategy

		return r
	}
}
