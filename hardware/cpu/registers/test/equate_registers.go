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

	"github.com/jetsetilly/gopher2600/hardware/cpu/registers"
)

// EquateRegisters is used to test equality between two instances of a register
// type. Used in testing packages.
func EquateRegisters(t *testing.T, value, expectedValue interface{}) {
	t.Helper()

	switch value := value.(type) {
	default:
		t.Fatalf("not a register type (%T)", value)

	case registers.Register:
		switch expectedValue := expectedValue.(type) {
		default:
			t.Fatalf("unhandled type (%T)", value)

		case int:
			if int(value.Value()) != expectedValue {
				t.Errorf("unexpected Register value (%#02x wanted %#02x)", value.Value(), expectedValue)
			}
		}

	case registers.ProgramCounter:
		switch expectedValue := expectedValue.(type) {
		default:
			t.Fatalf("unhandled type (%T)", value)

		case int:
			if int(value.Address()) != expectedValue {
				t.Errorf("unexpected ProgramCounter value (%#04x wanted %#04x)", value.Value(), expectedValue)
			}
		}

	case registers.StatusRegister:
		switch expectedValue := expectedValue.(type) {
		default:
			t.Fatalf("unhandled type (%T)", value)

		case int:
			if int(value.Value()) != expectedValue {
				t.Errorf("unexpected StatusRegister value (%#02x wanted %#02x)", value.Value(), expectedValue)
			}

		case string:
			if len(expectedValue) != 8 {
				t.Fatalf("status expressed as string must be 8 chars long")
			}
			if expectedValue[0] != 's' && !value.Sign || expectedValue[0] != 'S' && value.Sign {
				t.Errorf("unexpected StatusRegister flag (sign)")
			}
			if expectedValue[1] != 'v' && !value.Overflow || expectedValue[1] != 'V' && value.Overflow {
				t.Errorf("unexpected StatusRegister flag (overflow)")
			}
			if expectedValue[3] != 'b' && !value.Break || expectedValue[3] != 'B' && value.Break {
				t.Errorf("unexpected StatusRegister flag (break)")
			}
			if expectedValue[4] != 'd' && !value.DecimalMode || expectedValue[4] != 'D' && value.DecimalMode {
				t.Errorf("unexpected StatusRegister flag (decimal mode)")
			}
			if expectedValue[5] != 'i' && !value.InterruptDisable || expectedValue[5] != 'I' && value.InterruptDisable {
				t.Errorf("unexpected StatusRegister flag (interrupt diable)")
			}
			if expectedValue[6] != 'z' && !value.Zero || expectedValue[6] != 'Z' && value.Zero {
				t.Errorf("unexpected StatusRegister flag (zero)")
			}
			if expectedValue[7] != 'c' && !value.Carry || expectedValue[7] != 'C' && value.Carry {
				t.Errorf("unexpected StatusRegister flag (carry)")
			}
		}
	}
}
