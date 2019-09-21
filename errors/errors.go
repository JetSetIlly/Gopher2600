package errors

import (
	"fmt"
	"strings"
)

// Errno is used specified the specific error
type Errno int

// Values is the type used to specify arguments for FormattedErrors
type Values []interface{}

// AtariError provides a convenient way of providing arguments to a
// predefined error
type AtariError struct {
	Errno  Errno
	Values Values
}

// New is used to create a new instance of a FormattedError
func New(errno Errno, values ...interface{}) AtariError {
	er := new(AtariError)
	er.Errno = errno
	er.Values = values
	return *er
}

func (er AtariError) Error() string {
	s := fmt.Sprintf(messages[er.Errno], er.Values...)

	// de-duplicate error message parts
	p := strings.SplitN(s, ": ", 3)
	if len(p) > 1 && p[0] == p[1] {
		return strings.Join(p[1:], ": ")
	}

	return strings.Join(p, ": ")
}

// Is checks if most recently wrapped error is a AtariError with a specific errno
func Is(err error, errno Errno) bool {
	switch er := err.(type) {
	case AtariError:
		return er.Errno == errno
	}
	return false
}

// IsAny checks if most recently wrapped error is a AtariError with any errno
func IsAny(err error) bool {
	switch err.(type) {
	case AtariError:
		return true
	}
	return false
}

// Has checks to see if the specified AtariError appears somewhere in the
// sequence of wrapped errors
func Has(err error, errno Errno) bool {
	if Is(err, errno) {
		return true
	}

	for i := range err.(AtariError).Values {
		if e, ok := err.(AtariError).Values[i].(error); ok {
			if Has(e, errno) {
				return true
			}
		}
	}

	return false
}
