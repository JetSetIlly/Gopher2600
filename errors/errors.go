// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package errors

import (
	"fmt"
	"strings"
)

// Values is the type used to specify arguments for FormattedErrors
type Values []interface{}

// curated erorrs allow code to specify a predefined error and not worry too
// much about the message behind that error and how the message will be
// formatted on output.
type curated struct {
	message string
	values  Values
}

// Errorf creates a new curated error
func Errorf(message string, values ...interface{}) error {
	return curated{
		message: message,
		values:  values,
	}
}

// Error returns the normalised error message. Normalisation being the removal
// of duplicate adjacent error messsage parts.
//
// Implements the go language error interface.
func (er curated) Error() string {
	s := fmt.Errorf(er.message, er.values...).Error()

	// de-duplicate error message parts
	p := strings.SplitN(s, ": ", 3)
	if len(p) > 1 && p[0] == p[1] {
		return strings.Join(p[1:], ": ")
	}

	return strings.Join(p, ": ")
}

// Head returns the leading part of the message.
//
// Similar to Is() but returns the string rather than a boolean. Useful for
// switches.
//
// If err is a plain error then the return of Error() is returned
func Head(err error) string {
	if er, ok := err.(curated); ok {
		return er.message
	}
	return err.Error()
}

// IsAny checks if error is being curated by this package
func IsAny(err error) bool {
	if err == nil {
		return false
	}

	if _, ok := err.(curated); ok {
		return true
	}
	return false
}

// Is checks if error has a specific head
func Is(err error, head string) bool {
	if err == nil {
		return false
	}

	if er, ok := err.(curated); ok {
		return er.message == head
	}
	return false
}

// Has checks if the message string appears somewhere in the error
func Has(err error, msg string) bool {
	if err == nil {
		return false
	}

	if !IsAny(err) {
		return false
	}

	if Is(err, msg) {
		return true
	}

	for i := range err.(curated).values {
		if e, ok := err.(curated).values[i].(curated); ok {
			if Has(e, msg) {
				return true
			}
		}
	}

	return false
}
