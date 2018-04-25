package cpu_test

import (
	"fmt"
	"headlessVCS/hardware/memory"
	"testing"
)

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
	memory.CPUBus
}

func NewMockVCSMem() *MockVCSMem {
	mem := new(MockVCSMem)
	// use the memory.VCS implementation of memory.Bus
	mem.CPUBus = memory.NewVCSMemory()
	return mem
}

func (mem *MockVCSMem) putInstructions(origin uint16, bytes ...uint8) uint16 {
	for i, b := range bytes {
		mem.CPUBus.Write(uint16(i)+origin, b)
	}
	return origin + uint16(len(bytes))
}

func (mem *MockVCSMem) assert(t *testing.T, address uint16, value uint8) {
	t.Helper()
	d, _ := mem.CPUBus.Read(address)
	if d != value {
		t.Errorf("memory assertion failed (%v  - wanted %v at address %04x", d, value, address)
	}
}
