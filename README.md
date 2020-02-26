# retries

A simple, extensible Go retries library.

## Example

Without any arguments, the retrier uses the default strategy of using
exponential back-off, three retries, and retrying on all errors.

```go
myFunc := func() error {
	return errors.New("error")
}

retrier := New(
	func(retryNum int) error {
		log.Printf("retry number: %d", retryNum)
		return myFunc()
	},
)
if err := retrier.Try(); err != nil {
	log.Println(err)
}
```

All of the parameters in the default strategy can also be customized.
```go
myFunc := func() error {
	return errors.New("error")
}

retrier := New(
	func(retryNum int) error {
		log.Printf("retry number: %d", retryNum)
		return myFunc()
	},
	WithRetries(10),
	WithWhitelist(errors.New("error")),
	WithExpBackoff(2),
)
if err := retrier.Try(); err != nil {
	log.Println(err)
}
```
