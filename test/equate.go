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
	"testing"
)

// Equate is used to test equality between one value and another. Generally,
// both values must be of the same type but if a is of type uint16, b can be
// uint16 or int. The reason for this is that a literal number value is of type
// int. It is very convenient to write something like this, without having to
// cast the expected number value:
//
//	var r uint16
//	r = someFunction()
//	test.Equate(t, r, 10)
//
// This is by no means a comprehensive comparison function. With a bit more
// work with the reflect package we could generalise the testing a lot more. At
// is is however, it's good enough.
func Equate(t *testing.T, value, expectedValue interface{}) {
	t.Helper()

	switch v := value.(type) {
	default:
		t.Fatalf("unhandled type for Equate() function (%T))", v)

	case nil:
		if expectedValue != nil {
			t.Errorf("equation of type %T failed (%d  - wanted nil)", v, v)
		}

	case int64:
		switch ev := expectedValue.(type) {
		case int64:
			if v != ev {
				t.Errorf("equation of type %T failed (%d  - wanted %d)", v, v, expectedValue.(int64))
			}
		default:
			t.Fatalf("values for Equate() are not the same type (%T and %T)", v, expectedValue)
		}

	case uint64:
		switch ev := expectedValue.(type) {
		case uint64:
			if v != ev {
				t.Errorf("equation of type %T failed (%d  - wanted %d)", v, v, expectedValue.(uint64))
			}
		default:
			t.Fatalf("values for Equate() are not the same type (%T and %T)", v, expectedValue)
		}

	case int:
		switch ev := expectedValue.(type) {
		case int:
			if v != ev {
				t.Errorf("equation of type %T failed (%d  - wanted %d)", v, v, expectedValue.(int))
			}
		default:
			t.Fatalf("values for Equate() are not the same type (%T and %T)", v, expectedValue)
		}

	case uint16:
		switch ev := expectedValue.(type) {
		case int:
			if v != uint16(ev) {
				t.Errorf("equation of type %T failed (%d  - wanted %d)", v, v, ev)
			}
		case uint16:
			if v != ev {
				t.Errorf("equation of type %T failed (%d  - wanted %d)", v, v, ev)
			}
		default:
			t.Fatalf("values for Equate() are not the same compatible (%T and %T)", v, ev)
		}

	case string:
		switch ev := expectedValue.(type) {
		case string:
			if v != ev {
				t.Errorf("equation of type %T failed (%s  - wanted %s)", v, v, expectedValue.(string))
			}
		default:
			t.Fatalf("values for Equate() are not the same type (%T and %T)", v, expectedValue)
		}

	case bool:
		switch ev := expectedValue.(type) {
		case bool:
			if v != ev {
				t.Errorf("equation of type %T failed (%v  - wanted %v)", v, v, expectedValue.(bool))
			}
		default:
			t.Fatalf("values for Equate() are not the same type (%T and %T)", v, expectedValue)
		}
	}
}
