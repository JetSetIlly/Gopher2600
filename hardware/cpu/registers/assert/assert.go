package assert

import (
	"gopher2600/hardware/cpu/registers"
	"reflect"
	"testing"
)

// Assert is used to test equality between one value and another
func Assert(t *testing.T, r, x interface{}) {
	t.Helper()
	switch r := r.(type) {

	default:
		t.Errorf("assert failed (unknown type [%s])", reflect.TypeOf(r))

	case *registers.Register:
		switch x := x.(type) {
		default:
			t.Errorf("assert failed (unknown type [%s])", reflect.TypeOf(x))

		case int:
			if int(r.Value()) != x {
				t.Errorf("assert Register failed (%d  - wanted %d", r.Value(), x)
			}
		}

	case *registers.ProgramCounter:
		switch x := x.(type) {
		default:
			t.Errorf("assert failed (unknown type [%s])", reflect.TypeOf(x))

		case int:
			if int(r.Address()) != x {
				t.Errorf("assert ProgramCounter failed (%d  - wanted %d", r.Address(), x)
			}
		}

	case *registers.StatusRegister:
		switch x := x.(type) {
		default:
			t.Errorf("assert failed (unknown type [%s])", reflect.TypeOf(x))

		case int:
			if int(r.Value()) != x {
				t.Errorf("assert StatusRegister failed (%d  - wanted %d", r.Value(), x)
			}

		case string:
			if len(x) != 8 {
				t.Errorf("assert StatusRegister failed (status flags must be integer of a string of 8 chars)")
			}
			if x[0] != 's' && !r.Sign || x[0] != 'S' && r.Sign {
				t.Errorf("assert StatusRegister failed (unexpected sign flag")
			}
			if x[1] != 'v' && !r.Overflow || x[1] != 'V' && r.Overflow {
				t.Errorf("assert StatusRegister failed (unexpected overflow flag")
			}
			if x[3] != 'b' && !r.Break || x[3] != 'B' && r.Break {
				t.Errorf("assert StatusRegister failed (unexpected break flag")
			}
			if x[4] != 'd' && !r.DecimalMode || x[4] != 'D' && r.DecimalMode {
				t.Errorf("assert StatusRegister failed (unexpected decimal mode flag")
			}
			if x[5] != 'i' && !r.InterruptDisable || x[5] != 'I' && r.InterruptDisable {
				t.Errorf("assert StatusRegister failed (unexpected interrupt disable flag")
			}
			if x[6] != 'z' && !r.Zero || x[6] != 'Z' && r.Zero {
				t.Errorf("assert StatusRegister failed (unexpected zero flag")
			}
			if x[7] != 'c' && !r.Carry || x[7] != 'C' && r.Carry {
				t.Errorf("assert StatusRegister failed (unexpected carry flag")
			}
		}

	case uint16:
		switch x := x.(type) {
		default:
			t.Errorf("assert failed (unknown type [%s])", reflect.TypeOf(x))

		case int:
			if int(r) != x {
				t.Errorf("assert Register failed (%d  - wanted %d", r, x)
			}
		}

	case string:
		if r != x.(string) {
			t.Errorf("assert string failed (%v  - wanted %v", r, x.(string))
		}

	case bool:
		if r != x.(bool) {
			t.Errorf("assert bool failed (%v  - wanted %v", r, x.(bool))
		}

	case int:
		if r != x.(int) {
			t.Errorf("assert int failed (%d  - wanted %d)", r, x.(int))
		}
	}
}
