package cpu_test

// helpers_test.go contains the all the support code required for the cpu_test package
// it includes:
//
// o assert - used to test for equality between values
//
// o MockMem - a simple memory implementation satisfying the memory.Memory interface
//	- includes putInstructions(), a variadic function to place a sequence of bytes
//	into memory
//	- a clear method and and an assert method

import (
	"fmt"
	"headless/hardware/cpu"
	"testing"
)

func assert(t *testing.T, r, x interface{}) {
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
	data []uint8
}

func (mem *MockMem) String() string {
	s := "0000  "
	i := 0
	j := 16
	for _, d := range mem.data {
		s = fmt.Sprintf("%s%02x ", s, d)
		i++
		if i == 16 {
			s = fmt.Sprintf("%s\n%04d  ", s, j)
			i = 0
			j += 16
		}
	}
	return s
}

func NewMockMem() *MockMem {
	mock := new(MockMem)
	mock.data = make([]uint8, 512)
	return mock
}

func (mem *MockMem) Read(address uint16) (uint8, error) {
	if int(address) > len(mem.data) {
		return 0, fmt.Errorf("address out of range (%d)", address)
	}
	return mem.data[address], nil
}

func (mem *MockMem) Write(address uint16, data uint8) error {
	if int(address) > len(mem.data) {
		return fmt.Errorf("address out of range (%d)", address)
	}
	mem.data[address] = data
	return nil
}

func (mem *MockMem) clear() uint16 {
	fmt.Println("\nclearing memory\n---------------")
	for i := 0; i < len(mem.data); i++ {
		mem.data[i] = 0x00
	}
	return 0
}

func (mem *MockMem) putInstructions(origin uint16, bytes ...uint8) uint16 {
	for i, b := range bytes {
		mem.data[i+int(origin)] = b
	}
	return origin + uint16(len(bytes))
}

func (mem *MockMem) assert(t *testing.T, address uint16, value uint8) {
	t.Helper()
	if mem.data[address] != value {
		t.Errorf("assertMockMem failed (%v  - wanted %v at address %04x", mem.data[address], value, address)
	}
}
