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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package errors

import (
	"fmt"
	"strings"
)

// Values is the type used to specify arguments for FormattedErrors
type Values []interface{}

// AtariError allows code to specify a predefined error and not worry too much about the
// message behind that error and how the message will be formatted on output.
type AtariError struct {
	Head   string
	Values Values
}

// New is used to create a new instance of an AtariError.
func New(head string, values ...interface{}) AtariError {
	return AtariError{
		Head:   head,
		Values: values,
	}
}

// Error returns the normalised error message. Most usefully, it compresses
// duplicate adjacent AtariError instances.
func (er AtariError) Error() string {
	s := fmt.Sprintf(er.Head, er.Values...)

	// de-duplicate error message parts
	p := strings.SplitN(s, ": ", 3)
	if len(p) > 1 && p[0] == p[1] {
		return strings.Join(p[1:], ": ")
	}

	return strings.Join(p, ": ")
}

// Is checks if most recently wrapped error is an AtariError with a specific
// head
func Is(err error, head string) bool {
	switch er := err.(type) {
	case AtariError:
		return er.Head == head
	}
	return false
}

// IsAny checks if most recently wrapped error is an AtariError, with any head
func IsAny(err error) bool {
	switch err.(type) {
	case AtariError:
		return true
	}
	return false
}

// Has checks to see if the specified AtariError head appears somewhere in the
// sequence of wrapped errors
func Has(err error, head string) bool {
	if Is(err, head) {
		return true
	}

	for i := range err.(AtariError).Values {
		if e, ok := err.(AtariError).Values[i].(error); ok {
			if Has(e, head) {
				return true
			}
		}
	}

	return false
}
