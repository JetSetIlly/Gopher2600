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

package test

import (
	"math"
	"testing"
)

// ExpectEquality is used to test equality between one value and another
func ExpectEquality[T comparable](t *testing.T, value T, expectedValue T, tags ...any) bool {
	t.Helper()
	if value != expectedValue {
		t.Errorf("%sequality test of type %T failed: '%v' does not equal '%v')", id(tags...), value, value, expectedValue)
		return false
	}
	return true
}

// ExpectInequality is used to test inequality between one value and another. In
// other words, the test does not want to succeed if the values are equal
func ExpectInequality[T comparable](t *testing.T, value T, expectedValue T, tags ...any) bool {
	t.Helper()
	if value == expectedValue {
		t.Errorf("inequality test of type %T failed: '%v' does equal '%v')", value, value, expectedValue)
		return false
	}
	return true
}

// Approximate constraint used by ExpectApproximate() function
type Approximate interface {
	~float32 | ~float64 | ~int
}

// ExpectApproximate is used to test approximate equality between one value and
// another.
//
// Tolerance represents a percentage. For example, 0.5 is tolerance of +/- 50%.
// If the tolerance value is negative then the positive equivalent is used.
func ExpectApproximate[T Approximate](t *testing.T, value T, expectedValue T, tolerance float64, tags ...any) bool {
	t.Helper()

	tolerance = math.Abs(tolerance)

	top := float64(expectedValue) * (1 + tolerance)
	bot := float64(expectedValue) * (1 - tolerance)

	if float64(value) < bot || float64(value) > top {
		t.Errorf("%sapproximation test of type %T failed: '%v' is outside the range '%v' to '%v')", id(tags...), value, value, top, bot)
		return false
	}
	return true
}

// ExpectFailure tests for an 'unsucessful value for the value's type.
//
// Types bool and error are treated thus:
//
//	bool == false
//	error != nil
//
// # If type is nil then the test will fail
//
// Any other type will fatally fail - only bool, error and nil are supported
func ExpectFailure(t *testing.T, v any, tags ...any) bool {
	t.Helper()
	if expect(t, v) {
		t.Errorf("%sa failure value is expected for type %T", id(tags...), v)
		return false
	}
	return true
}

// ExpectSuccess tests for a 'sucessful' value for the value's type.
//
// Types bool and error are treated thus:
//
//	bool == true
//	error == nil
//
// # If type is nil then the test will succeed
//
// Any other type will fatally fail - only bool, error and nil are supported
func ExpectSuccess(t *testing.T, v any, tags ...any) bool {
	t.Helper()
	if !expect(t, v) {
		t.Errorf("%sa success value is expected for type %T", id(tags...), v)
		return false
	}
	return true
}

// expect is a basic test for success/failure. if it returns true the value v is
// considered to be "successful"
func expect(t *testing.T, v any, tags ...any) bool {
	t.Helper()

	switch v := v.(type) {
	case bool:
		if !v {
			return false
		}

	case error:
		if v != nil {
			return false
		}

	case nil:
		return true

	default:
		t.Fatalf("%sunsupported type (%T) for expect test", id(tags...), v)
	}

	return true
}
