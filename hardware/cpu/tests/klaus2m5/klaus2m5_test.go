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

package klaus2m5

import (
	_ "embed"
	"testing"

	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/test"
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

func (mem *testMem) Write(address uint16, data uint8) error {
	mem.internal[address] = data
	return nil
}

type checkResult int

const (
	running checkResult = iota
	success
	fail
)

type opts struct {
	bin    []byte
	origin uint16
	entry  uint16
	check  func(*cpu.CPU, *testMem) checkResult
}

func runTestBinary(t *testing.T, opt opts) {
	mem := newTestMem()
	copy(mem.internal[opt.origin:], opt.bin)

	// set reset vectors
	mem.internal[cpu.Reset] = byte(opt.entry)
	mem.internal[cpu.Reset+1] = byte(opt.entry >> 8)

	// create CPU. reset will be done in run() function
	mc := cpu.NewCPU(mem)

	// cpu snapshot to be examined in case of test failure
	type snapshot struct {
		mc    *cpu.CPU
		stack []byte
	}
	var history [5]snapshot

	// the run function is executed at least once with the trace parameter set to
	// false. if the run() fails, the function is run again with the trace
	// parameter set to true
	run := func(trace bool) bool {
		err := mc.Reset(nil)
		if err != nil {
			t.Fatal(err)
		}

		for {
			err := mc.ExecuteInstruction(cpu.NilCycleCallback)
			if err != nil {
				t.Fatal(err)
			}

			if trace {
				copy(history[:], history[1:])
				history[len(history)-1].mc = mc.Snapshot()
				history[len(history)-1].stack = mem.internal[0x0100|mc.SP.Address()+1 : 0x0200]
			}

			switch opt.check(mc, mem) {
			case success:
				return true
			case fail:
				return false
			}
		}
	}

	if !run(false) {
		// the first run() failed so we run it again with the trace parameter
		// set to true. note that we expect the execution to return false. if it
		// does not then something unexpected has gone wrong
		ok := run(true)
		test.DemandFailure(t, ok)

		// output immediate CPU history
		for _, l := range history {
			if l.mc != nil {
				t.Logf("%s (opcode %02x)", l.mc.LastResult.String(), l.mc.LastResult.Defn.OpCode)
				t.Logf("%s", l.mc.String())
				if len(l.stack) == 0 {
					t.Log("[stack is empty]")
				} else {
					t.Logf("[stack % 02x]", l.stack)
				}
			}
		}

		t.Fail()
	}
}

//go:embed "functional_test/6502_functional_test.bin"
var functionalTest []byte

func TestFunctional(t *testing.T) {
	var loopCt int

	runTestBinary(t, opts{
		bin: functionalTest,

		// the origin address is defined by the "zero_page" value in the source file
		origin: uint16(0x000a),

		// the entry is defined by the "code_segment" value in the source file
		entry: uint16(0x0400),

		check: func(mc *cpu.CPU, mem *testMem) checkResult {
			// this test succeeds when it reaches a jmp instruction that jumps to it's own address.
			// ie. an exceedingly tight infinite loop

			// both success and fail happen when a JMP instruction jumps to itself causing an
			// infinite loop. we first detect this loop and then check for the success address
			a := mc.PC.Address()
			if mc.LastResult.Address == a {
				loopCt++
				if loopCt >= 10 {
					// success address from functional_test/6502_functional_test.lst
					if a == 0x347d {
						return success
					}
					return fail
				}
			} else {
				loopCt = 0
			}
			return running
		},
	})
}

//go:embed "decimal_mode/6502_decimal_test.bin"
var decimalModeTest []byte

func TestDecimalMode(t *testing.T) {
	runTestBinary(t, opts{
		bin:    decimalModeTest,
		origin: uint16(0x0200),
		entry:  uint16(0x0200),
		check: func(mc *cpu.CPU, mem *testMem) checkResult {
			if mc.LastResult.Defn.OpCode == 0xdb {
				b, _ := mem.Read(0x000b)
				if b == 0x00 {
					return success
				}
				return fail
			}
			return running
		},
	})
}
