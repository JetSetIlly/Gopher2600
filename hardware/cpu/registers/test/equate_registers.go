package test

import (
	"gopher2600/hardware/cpu/registers"
	"testing"
)

// EquateRegisters is used to test equality between two instances of a register
// type. Used in testing packages.
func EquateRegisters(t *testing.T, value, expectedValue interface{}) {
	t.Helper()

	switch value := value.(type) {

	default:
		t.Fatalf("not a register type (%T)", value)

	case *registers.Register:
		switch expectedValue := expectedValue.(type) {
		default:
			t.Fatalf("unhandled type (%T)", value)

		case int:
			if int(value.Value()) != expectedValue {
				t.Errorf("unexpected Register value (%d wanted %d)", value.Value(), expectedValue)
			}
		}

	case *registers.ProgramCounter:
		switch expectedValue := expectedValue.(type) {
		default:
			t.Fatalf("unhandled type (%T)", value)

		case int:
			if int(value.Address()) != expectedValue {
				t.Errorf("unexpected ProgramCounter value (%d wanted %d)", value, expectedValue)
			}
		}

	case *registers.StatusRegister:
		switch expectedValue := expectedValue.(type) {
		default:
			t.Fatalf("unhandled type (%T)", value)

		case int:
			if int(value.Value()) != expectedValue {
				t.Errorf("unexpected StatusRegister value (%d wanted %d)", value.Value(), expectedValue)
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
