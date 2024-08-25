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

// DemandEquility is used to test equality between one value and another. If the
// test fails it is a testing fatility
//
// This is particular useful if the values being tested are used in further
// tests and so must be correct. For example, testing that the lengths of two
// slices are equal before iterating over them in unison
func DemandEquality[T comparable](t *testing.T, value T, expectedValue T) {
	t.Helper()
	if value != expectedValue {
		t.Fatalf("equality test of type %T failed: '%v' does not equal '%v')", value, value, expectedValue)
	}
}
