package errors

import "fmt"

// Errno is used specified the specific error
type Errno int

// Values is the type used to specify arguments for FormattedErrors
type Values []interface{}

// FormattedError provides a convenient way of providing arguments to a
// predefined error
type FormattedError struct {
	Errno  Errno
	Values Values
}

// NewFormattedError is used to create a new instance of a FormattedError
func NewFormattedError(errno Errno, values ...interface{}) FormattedError {
	er := new(FormattedError)
	er.Errno = errno
	er.Values = values
	return *er
}

func (er FormattedError) Error() string {
	return fmt.Sprintf(messages[er.Errno], er.Values...)
}
