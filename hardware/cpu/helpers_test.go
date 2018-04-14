package cpu_test

// helpers_test.go contains the all the support code required for the cpu_test package
// it includes:
//
// o assert - used to test for equality between values
//
// o MockMem - implementation of memory.Bus
//	- embeds the VCS implementation of memory.Bus for convenience
//	- plus:
//		- putInstructions(), a variadic function to place a sequence of bytes into memory
//		- an assert method

import (
	"fmt"
	"headless/hardware/cpu"
	"headless/hardware/memory"
	"testing"
)

func assert(t *testing.T, r, x interface{}) {
	t.Helper()
	switch r := r.(type) {
	case cpu.StatusRegister:
		if fmt.Sprintf("%s", r) != x.(string) {
			t.Errorf("assert StatusRegister failed (%s  - wanted %s)", r, x.(string))
		}
	case cpu.Register:
		switch x := x.(type) {
		case int:
			if r.ToUint16() != uint16(x) {
				t.Errorf("assert Register failed (%d  - wanted %d", r.ToUint16(), x)
			}
		case string:
			if r.ToBits() != x {
				t.Errorf("assert Register failed (%s  - wanted %s", r.ToBits(), x)
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

// MockMem is an extenstion of the memory.Bus interface
type MockMem struct {
	memory.Bus
}

func NewMockMem() *MockMem {
	mem := new(MockMem)
	// use the memory.VCS implementation of memory.Bus
	mem.Bus = memory.NewVCSMemory()
	return mem
}

func (mem *MockMem) putInstructions(origin uint16, bytes ...uint8) uint16 {
	for i, b := range bytes {
		mem.Bus.Write(uint16(i)+origin, b)
	}
	return origin + uint16(len(bytes))
}

func (mem *MockMem) assert(t *testing.T, address uint16, value uint8) {
	t.Helper()
	d, _ := mem.Bus.Read(address)
	if d != value {
		t.Errorf("assertMockMem failed (%v  - wanted %v at address %04x", d, value, address)
	}
}
