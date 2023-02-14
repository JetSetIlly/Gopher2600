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

//go:build 6507_functional_test

package functional_test

import (
	_ "embed"
	"testing"

	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
)

type testMem struct {
	internal []uint8
}

func newTestMem() *testMem {
	return &testMem{
		internal: make([]uint8, 0x10000),
	}
}

func (mem testMem) Read(address uint16) (uint8, error) {
	return mem.internal[address], nil
}

func (mem testMem) ReadZeroPage(address uint8) (uint8, error) {
	return mem.Read(uint16(address))
}

func (mem *testMem) Write(address uint16, data uint8) error {
	mem.internal[address] = data
	return nil
}

//go:embed "6502_functional_test.bin"
var functionalTest []byte

func TestFunctional(t *testing.T) {
	var programOrigin = uint16(0x0400)
	var loadAddress = uint16(0x000a)
	var successAddress = uint16(0x347d)

	mem := newTestMem()
	copy(mem.internal[loadAddress:], functionalTest)

	// set reset vectors
	mem.internal[cpubus.Reset] = byte(programOrigin)
	mem.internal[cpubus.Reset+1] = byte(programOrigin >> 8)

	mc := cpu.NewCPU(nil, mem)
	mc.Reset()
	mc.LoadPCIndirect(cpubus.Reset)

	// mc.ExecutionInstruction() requires a callback function even if it does
	// nothing
	callback := func() error {
		return nil
	}

	// cpu history to be examined in case of test failure
	type history struct {
		mc    *cpu.CPU
		stack []byte
	}
	var lastResult [15]history

	var success bool

	for {
		addr := mc.PC.Address()

		err := mc.ExecuteInstruction(callback)
		if err != nil {
			t.Fatal(err)
		}

		copy(lastResult[:], lastResult[1:])
		lastResult[len(lastResult)-1].mc = mc.Snapshot()
		lastResult[len(lastResult)-1].stack = mem.internal[0x0100|mc.SP.Address()+1 : 0x0200]

		// reaching the successAddress means that all tests have completed
		if mc.PC.Address() == successAddress || mc.PC.Address() == programOrigin {
			success = true
			break // for loop
		}

		// "Loop on program counter determines error or successful completion of test"
		if mc.PC.Address() == addr {
			success = false
			break // for loop
		}
	}

	// output immediate CPU history if test fails
	if !success {
		for _, l := range lastResult {
			if l.mc != nil {
				t.Logf("%s (opcode %02x)", l.mc.LastResult.String(), l.mc.LastResult.Defn.OpCode)
				t.Logf("%s", l.mc.String())
				if len(l.stack) == 0 {
					t.Log("[stack is empty]")
				} else {
					t.Logf("[% 02x]", l.stack)
				}
			}
		}
		t.Fail()
	}
}
