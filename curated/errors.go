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

package curated

import (
	"fmt"
	"strings"
)

// curated is an implementation of the go language error interface.
type curated struct {
	pattern string
	values  []interface{}
}

// Errorf creates a new curated error.
//
// Note that unlike the Errorf() function in the fmt package the first argument
// is named "pattern" not "format". This is because we use the pattern string
// in the Is() and Has() functions where 'pattern' seems to be more descriptive
// name.
func Errorf(pattern string, values ...interface{}) error {
	// note that we're not actually formatting the error here, despite the
	// function name. we instead only store the arguments. formatting takes
	// place in the Error() function
	return curated{
		pattern: pattern,
		values:  values,
	}
}

// Error returns the normalised error message. Normalisation being the removal
// of duplicate adjacent error messsage parts in the error message chains. It
// doesn't affect letter-case or white space.
//
// Implements the go language error interface.
func (er curated) Error() string {
	s := fmt.Errorf(er.pattern, er.values...).Error()

	// de-duplicate error message parts
	p := strings.SplitN(s, ": ", 3)
	if len(p) > 1 && p[0] == p[1] {
		return strings.Join(p[1:], ": ")
	}

	return strings.Join(p, ": ")
}

// IsAny checks if the error is a curated error.
func IsAny(err error) bool {
	if err == nil {
		return false
	}

	if _, ok := err.(curated); ok {
		return true
	}

	return false
}

// Is checks if error is a curated error with a specific pattern.
func Is(err error, pattern string) bool {
	if err == nil {
		return false
	}

	if er, ok := err.(curated); ok {
		return er.pattern == pattern
	}

	return false
}

// Is checks if error is a curated error with a specific pattern somewhere in
// the chain.
func Has(err error, pattern string) bool {
	if err == nil {
		return false
	}

	if !IsAny(err) {
		return false
	}

	if Is(err, pattern) {
		return true
	}

	for i := range err.(curated).values {
		if e, ok := err.(curated).values[i].(curated); ok {
			if Has(e, pattern) {
				return true
			}
		}
	}

	return false
}
