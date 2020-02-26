# retries

A simple, extensible Go retries library.

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
