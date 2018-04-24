package registers_test

import (
	"headlessVCS/hardware/cpu/registers"
	"testing"
)

func assert(t *testing.T, r, x interface{}) {
	t.Helper()
	switch r := r.(type) {
	case registers.Bits:
		switch x := x.(type) {
		case int:
			if r.ToUint16() != uint16(x) {
				t.Errorf("assert Register failed (%d  - wanted %d", r.ToUint16(), x)
			}
		case string:
			if r.ToString() != x {
				t.Errorf("assert Register failed (%s  - wanted %s", r.ToString(), x)
			}
		}
	case bool:
		if r != x.(bool) {
			t.Errorf("assert Bool failed (%v  - wanted %v", r, x.(bool))
		}
	case int:
		if r != x.(int) {
			t.Errorf("assert Int failed (%d  - wanted %d)", r, x.(int))
		}
	}
}
