package errors

import "fmt"

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
	return fmt.Sprintf(messages[er.Errno], er.Values...)
}

// Is checks if error is a AtariError with a specific errno
func Is(err error, errno Errno) bool {
	switch er := err.(type) {
	case AtariError:
		return er.Errno == errno
	}
	return false
}

// IsAny checks if error is a AtariError with any errno
func IsAny(err error) bool {
	switch err.(type) {
	case AtariError:
		return true
	}
	return false
}
