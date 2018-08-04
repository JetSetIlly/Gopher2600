package errors

import "fmt"

// Errno is used specified the specific error
type Errno int

// Values is the type used to specify arguments for a GopherError
type Values []interface{}

// GopherError is the error type used by Gopher2600
type GopherError struct {
	Errno  Errno
	Values Values
}

// NewGopherError is used to create a Gopher2600 specific error
func NewGopherError(errno Errno, values ...interface{}) GopherError {
	ge := new(GopherError)
	ge.Errno = errno
	ge.Values = values
	return *ge
}

func (er GopherError) Error() string {
	return fmt.Sprintf(messages[er.Errno], er.Values...)
}
