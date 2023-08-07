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

import "testing"

// ExpectEquality is used to test equality between one value and another
func ExpectEquality[T comparable](t *testing.T, value T, expectedValue T) {
	t.Helper()
	if value != expectedValue {
		t.Errorf("equality test of type %T failed: %v does not equal %v)", value, value, expectedValue)
	}
}

// ExpectInequality is used to test inequality between one value and another. In
// other words, the test does not want to succeed if the values are equal
func ExpectInequality[T comparable](t *testing.T, value T, expectedValue T) {
	t.Helper()
	if value == expectedValue {
		t.Errorf("inequality test of type %T failed: %v does equal %v)", value, value, expectedValue)
	}
}

// ExpectFailure tests argument v for a failure condition suitable for it's
// type. Types bool and error are treated thus:
//
//	bool == false
//	error != nil
//
// If type is nil then the test will fail
func ExpectFailure(t *testing.T, v interface{}) bool {
	t.Helper()

	switch v := v.(type) {
	case bool:
		if v {
			t.Errorf("failure test of type %T failed: %T does not equal false", v, v)
			return false
		}

	case error:
		if v == nil {
			t.Errorf("failure test of type %T failed: %T equals nil", v, v)
			return false
		}

	case nil:
		t.Errorf("failure test of type %T failed: %T is not expected", v, v)
		return false

	default:
		t.Fatalf("unsupported type %T for ExpectFailure()", v)
		return false
	}

	return true
}

// ExpectSuccess tests argument v for a success condition suitable for it's
// type. Types bool and error are treated thus:
//
//	bool == true
//	error == nil
//
// If type is nil then the test will succeed
func ExpectSuccess(t *testing.T, v interface{}) bool {
	t.Helper()

	switch v := v.(type) {
	case bool:
		if !v {
			t.Errorf("success test of type %T failed: %T does not equal true", v, v)
			return false
		}

	case error:
		if v != nil {
			t.Errorf("success test of type %T failed: %T does not equal nil", v, v)
			return false
		}

	case nil:
		return true

	default:
		t.Fatalf("unsupported type (%T) for ExpectSuccess()", v)
		return false
	}

	return true
}
