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

type MockMem struct {
	internal []uint8
}

func NewMockMem() *MockMem {
	mem := new(MockMem)
	mem.internal = make([]uint8, 0x10000)
	return mem
}

func (mem *MockMem) putInstructions(origin uint16, bytes ...uint8) uint16 {
	for i, b := range bytes {
		mem.Write(uint16(i)+origin, b)
	}
	return origin + uint16(len(bytes))
}

func (mem *MockMem) assert(t *testing.T, address uint16, value uint8) {
	t.Helper()
	d, _ := mem.Read(address)
	if d != value {
		t.Errorf("memory assertion failed (%v  - wanted %v at address %04x", d, value, address)
	}
}

// Clear sets all bytes in memory to zero
func (mem *MockMem) Clear() {
	for i := 0; i < len(mem.internal); i++ {
		mem.internal[i] = 0
	}
}

func (mem *MockMem) Read(address uint16) (uint8, error) {
	if int(address) > len(mem.internal) {
		return 0, fmt.Errorf("address out of range (%d)", address)
	}
	return mem.internal[address], nil
}

func (mem *MockMem) Write(address uint16, data uint8) error {
	if int(address) > len(mem.internal) {
		return fmt.Errorf("address out of range (%d)", address)
	}
	mem.internal[address] = data
	return nil
}

// MockVCSMem is an extenstion of the memory.Bus interface
type MockVCSMem struct {
	memory.Bus
}

func NewMockVCSMem() *MockVCSMem {
	mem := new(MockVCSMem)
	// use the memory.VCS implementation of memory.Bus
	mem.Bus = memory.NewVCSMemory()
	return mem
}

func (mem *MockVCSMem) putInstructions(origin uint16, bytes ...uint8) uint16 {
	for i, b := range bytes {
		mem.Bus.Write(uint16(i)+origin, b)
	}
	return origin + uint16(len(bytes))
}

func (mem *MockVCSMem) assert(t *testing.T, address uint16, value uint8) {
	t.Helper()
	d, _ := mem.Bus.Read(address)
	if d != value {
		t.Errorf("memory assertion failed (%v  - wanted %v at address %04x", d, value, address)
	}
}
