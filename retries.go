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
	conf struct {
		sleeper        Sleeper
		retries        int
		errorProcessor func(error) error
		retryCheck     func(error) bool
		retrySleeper   func(int, Sleeper)
	}

	sleeper struct{}

	Sleeper interface {
		Sleep(time.Duration)
	}

	Arg func(*conf) *conf

	Func func(int) error

	Retryer struct {
		f    Func
		conf *conf
	}
)

func New(f Func, args ...Arg) *Retryer {
	conf := &conf{}
	for _, a := range args {
		a(conf)
	}

	conf.setDefaults()

	return &Retryer{
		f:    f,
		conf: conf,
	}
}

func (r *Retryer) Try() error {
	var err error

	for i := 0; i < r.conf.retries; i++ {
		err = r.f(i)
		if err == nil {
			return nil
		}

		if i != r.conf.retries-1 {
			err = r.conf.errorProcessor(err)
			if r.conf.retryCheck(err) {
				r.conf.retrySleeper(i, r.conf.sleeper)
			} else {
				break
			}
		}
	}

	return err
}

func (c *conf) setDefaults() {
	if c.sleeper == nil {
		c.sleeper = &sleeper{}
	}
	if c.retries == 0 {
		c.retries = defaultRetries
	}
	if c.errorProcessor == nil {
		c.errorProcessor = defaultErrorProcesor
	}
	if c.retryCheck == nil {
		c.retryCheck = defaultRetryCheck
	}
	if c.retrySleeper == nil {
		WithExpBackoff(defaultBackoffFactor)(c)
	}
}

func (s *sleeper) Sleep(d time.Duration) {
	time.Sleep(d)
}

func defaultErrorProcesor(err error) error {
	return err
}

func defaultRetryCheck(err error) bool {
	return err != nil
}

func WithExpBackoff(factor int) Arg {
	return func(c *conf) *conf {
		c.retrySleeper = func(retryNum int, sleeper Sleeper) {
			s := math.Pow(float64(factor), float64(retryNum))
			sleeper.Sleep(time.Second * time.Duration(s))
		}

		return c
	}
}

func WithWhitelist(whitelist ...error) Arg {
	return func(c *conf) *conf {
		c.retryCheck = func(err error) bool {
			for _, e := range whitelist {
				if errors.Is(err, e) {
					return true
				}
			}

			return false
		}

		return c
	}
}

func WithBlacklist(blacklist ...error) Arg {
	return func(c *conf) *conf {
		c.retryCheck = func(err error) bool {
			for _, e := range blacklist {
				if errors.Is(err, e) {
					return false
				}
			}

			return true
		}

		return c
	}
}
