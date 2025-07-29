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

package functional_test

import (
	_ "embed"
	"os"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/test"
)

const (
	// whether to create a CPU profile of the host computer when running the test
	profiling = true

	// whether to test the approximate FPS against the expected FPS value
	approximationTest = false
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

// these addresses are specific to the functional test binary
var programOrigin = uint16(0x0400)
var loadAddress = uint16(0x000a)
var successAddress = uint16(0x347d)

func TestFunctional(t *testing.T) {
	mem := newTestMem()
	copy(mem.internal[loadAddress:], functionalTest)

	// set reset vectors
	mem.internal[cpu.Reset] = byte(programOrigin)
	mem.internal[cpu.Reset+1] = byte(programOrigin >> 8)

	// create CPU. reset will be done in run() function
	mc := cpu.NewCPU(mem)

	// cpu snapshot to be examined in case of test failure
	type snapshot struct {
		mc    *cpu.CPU
		stack []byte
	}
	var history [15]snapshot

	// benchmarking. reset on every call to run()
	var totalCycles int
	var startTime time.Time

	// the run function is run at least once with the record parameter set to
	// false. if the run() fails, the function is run again with the record
	// parameter set to true
	run := func(record bool) bool {
		// start and end profile only if record is set to false - we don't want
		// to profile all the memory allocations
		if profiling && !record {
			f, err := os.Create("cpu_performance.profile")
			if err != nil {
				t.Fatal(err.Error())
			}
			defer func() {
				err := f.Close()
				if err != nil {
					t.Fatal(err.Error())
				}
			}()

			err = pprof.StartCPUProfile(f)
			if err != nil {
				t.Fatal(err.Error())
			}
			defer pprof.StopCPUProfile()
		}

		totalCycles = 0
		startTime = time.Now()

		err := mc.Reset(nil)
		if err != nil {
			t.Fatal(err)
		}

		for {
			addr := mc.PC.Address()

			err := mc.ExecuteInstruction(cpu.NilCycleCallback)
			if err != nil {
				t.Fatal(err)
			}

			totalCycles += mc.LastResult.Cycles

			if record {
				copy(history[:], history[1:])
				history[len(history)-1].mc = mc.Snapshot()
				history[len(history)-1].stack = mem.internal[0x0100|mc.SP.Address()+1 : 0x0200]
			}

			// reaching the successAddress means that all tests have completed
			if mc.PC.Address() == successAddress || mc.PC.Address() == programOrigin {
				return true
			}

			// "Loop on program counter determines error or successful completion of test"
			if mc.PC.Address() == addr {
				return false
			}
		}
	}

	if run(false) {
		// approximate FPS assuming the frame generated is a standard NTSC image
		frames := totalCycles / (specification.SpecNTSC.ScanlinesTotal * specification.ClksScanline)
		fps := frames / int(time.Since(startTime).Seconds())
		t.Logf("approx FPS: %d", fps)

		// the totalCycles and frames value are the same regardless of the
		// performance and capabilities of the host machine
		test.ExpectEquality(t, totalCycles, 96247556)
		test.ExpectEquality(t, frames, 1611)

		// approximation test for fps value
		//
		// not really useful because the results depend on the underlying hardware.
		// one way of improving this is to use some sort of CPU ID package that
		// sets the expected value according to the detected CPU
		if approximationTest {
			test.ExpectApproximate(t, fps, 268, 0.1)
		}
	} else {
		// the first run() failed so we run it again with the record parameter
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
					t.Logf("[% 02x]", l.stack)
				}
			}
		}
		t.Fail()
	}
}
