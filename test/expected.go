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

// ExpectedFailure tests argument v for a failure condition suitable for it's
// type. Currentlly support types:
//
//		bool -> bool == false
//		error -> error != nil
//
// If type is nil then the test will fail.
func ExpectedFailure(t *testing.T, v interface{}) bool {
	t.Helper()

	switch v := v.(type) {
	case bool:
		if v {
			t.Errorf("expected failure (bool)")
			return false
		}

	case error:
		if v == nil {
			t.Errorf("expected failure (error)")
			return false
		}

	case nil:
		t.Errorf("expected failure (nil)")
		return false

	default:
		t.Fatalf("unsupported type (%T) for expectation testing", v)
		return false
	}

	return true
}

// ExpectedSuccess tests argument v for a success condition suitable for it's
// type. Currentlly support types:
//
//		bool -> bool == true
//		error -> error == nil
//
// If type is nil then the test will succeed.
func ExpectedSuccess(t *testing.T, v interface{}) bool {
	t.Helper()

	switch v := v.(type) {
	case bool:
		if !v {
			t.Errorf("expected success (bool)")
			return false
		}

	case error:
		if v != nil {
			t.Errorf("expected success (error: %v)", v)
			return false
		}

	case nil:
		return true

	default:
		t.Fatalf("unsupported type (%T) for expectation testing", v)
		return false
	}

	return true
}
