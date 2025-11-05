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

package thomharte

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/test"
)

// the posible memory events recorded by the memory implementation. also used to seal the memEvent
// types in the BusCycle test data
type memEvent string

const (
	read  = memEvent("read")
	write = memEvent("write")
)

type testMem struct {
	internal   []uint8
	addressBus uint16
	dataBus    uint8
	lastEvent  memEvent
}

func newTestMem() *testMem {
	return &testMem{
		// the CPU has a 16bit address bus so the maximum amount of memory is 64k
		internal: make([]uint8, 0x10000),
	}
}

func (mem *testMem) Read(address uint16) (uint8, error) {
	mem.addressBus = address
	mem.dataBus = mem.internal[address]
	mem.lastEvent = read
	return mem.dataBus, nil
}

func (mem *testMem) Write(address uint16, data uint8) error {
	mem.addressBus = address
	mem.dataBus = data
	mem.internal[address] = data
	mem.lastEvent = write
	return nil
}

type RAMEntry struct {
	Address uint16 `json:"0"`
	Value   uint8  `json:"1"`
}

func (r *RAMEntry) UnmarshalJSON(data []byte) error {
	var raw [2]uint64
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}
	r.Address = uint16(raw[0])
	r.Value = uint8(raw[1])
	return nil
}

type BusCycle struct {
	Address uint16
	Data    uint8
	Event   memEvent
}

func (b *BusCycle) UnmarshalJSON(data []byte) error {
	var raw [3]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	addr, _ := raw[0].(float64)
	dat, _ := raw[1].(float64)
	ev, _ := raw[2].(string)

	b.Address = uint16(addr)
	b.Data = uint8(dat)
	b.Event = memEvent(ev)

	switch b.Event {
	case read, write:
	default:
		return fmt.Errorf("unexpected memory event: %q", b.Event)
	}

	return nil
}

type State struct {
	PC  uint64     `json:"pc"`
	S   uint64     `json:"s"`
	A   uint64     `json:"a"`
	X   uint64     `json:"x"`
	Y   uint64     `json:"y"`
	P   uint64     `json:"p"`
	RAM []RAMEntry `json:"ram"`
}

type Tests struct {
	Name    string     `json:"name"`
	Initial State      `json:"initial"`
	Final   State      `json:"final"`
	Cycles  []BusCycle `json:"cycles"`
}

func (d *Tests) UnmarshalJSON(data []byte) error {
	// we have a custom unmarshaller for Tests only so that we can insert the Name field to any
	// error. to make the unmarshaller as clean as possible we want to avoid recursion; and we can
	// do this by using an alias type
	type norecurse Tests

	var tmp norecurse
	if err := json.Unmarshal(data, &tmp); err != nil {
		return fmt.Errorf("error unmarshalling test %q: %w", tmp.Name, err)
	}
	*d = Tests(tmp)
	return nil
}

var testsPath = filepath.Join("6502", "v1")

func TestThomHarte(t *testing.T) {
	var env = os.Getenv("GOPHER2600_SINGLESTEP_TEST")
	if len(env) == 0 {
		return
	}

	selected := strings.Split(env, ",")
	for _, s := range selected {
		rng := strings.SplitN(s, "-", 2)
		switch len(rng) {
		case 1:
			n, err := strconv.ParseUint(rng[0], 16, 8)
			if err != nil {
				t.Fatalf("opcode is malformed: %s: %v", s, err)
			}
			testThomHarte(t, uint8(n), true)
		case 2:
			n, err := strconv.ParseUint(rng[0], 16, 8)
			if err != nil {
				t.Fatalf("opcode range is malformed: %s: %v", s, err)
			}
			e, err := strconv.ParseUint(rng[1], 16, 8)
			if err != nil {
				t.Fatalf("opcode range is malformed: %s: %v", s, err)
			}
			if n >= e {
				t.Fatalf("opcode ranges should run from low to high: ie. %02x-%02x not %s", e, n, s)
			}
			for n <= e {
				testThomHarte(t, uint8(n), false)
				n++
			}
		default:
			t.Fatalf("opcode is malformed: %s", s)
		}
	}
}

// 0xab (LAX immediate) is working but the "internal parameter" we're using in the cpu package is
// not what the tests expect. the instruction is therefore disabled until we add the option to set
// the internal parameter manually
var notWorking = []uint8{0x93, 0x9f, 0x9b, 0x9c, 0x9e, 0xab}

func testThomHarte(t *testing.T, opcode uint8, force bool) {
	testFile := filepath.Join(testsPath, fmt.Sprintf("%02x.json", opcode))

	if slices.Contains(notWorking, opcode) {
		if force {
			t.Logf("forcing %s", testFile)
		} else {
			t.Logf("skipping %s", testFile)
			return
		}
	} else {
		t.Logf("testing %s", testFile)
	}

	f, err := os.Open(testFile)
	if err != nil {
		t.Fatal(err)
	}

	var tests []Tests
	if err := json.NewDecoder(f).Decode(&tests); err != nil {
		t.Fatalf("%s: %v", testFile, err)
	}

	mem := newTestMem()
	mc := cpu.NewCPU(mem)

	for i, s := range tests {
		mc.Reset(nil)

		fatalTest := func() {
			debug.PrintStack()
			t.Logf("last instruction: %s", mc.LastResult.Defn.String())
			t.Fatalf("%s: failed on line %d, cycle %d", testFile, i, mc.LastResult.Cycles-1)
		}

		mc.PC.Load(uint16(s.Initial.PC))
		mc.A.Load(uint8(s.Initial.A))
		mc.X.Load(uint8(s.Initial.X))
		mc.Y.Load(uint8(s.Initial.Y))
		mc.SP.Load(uint8(s.Initial.S))
		mc.Status.Load(uint8(s.Initial.P))
		for _, r := range s.Initial.RAM {
			mem.internal[r.Address] = r.Value
		}

		hook := func() error {
			cycle := mc.LastResult.Cycles - 1

			var fail bool

			fail = !test.ExpectEquality(t, mem.addressBus, s.Cycles[cycle].Address,
				testFile, i, "address bus cycle", cycle) || fail
			fail = !test.ExpectEquality(t, mem.dataBus, s.Cycles[cycle].Data,
				testFile, i, "data bus cycle", cycle) || fail
			fail = !test.ExpectEquality(t, mem.lastEvent, s.Cycles[cycle].Event,
				testFile, i, "memory event cycle", cycle) || fail

			if fail {
				fatalTest()
			}

			return nil
		}

		err := mc.ExecuteInstruction(hook)
		if err != nil {
			t.Fatal(err)
		}

		var fail bool

		fail = !test.ExpectEquality(t, mc.PC.Value(), uint16(s.Final.PC), testFile, i, "PC") || fail
		fail = !test.ExpectEquality(t, mc.A.Value(), uint8(s.Final.A), testFile, i, "A") || fail
		fail = !test.ExpectEquality(t, mc.X.Value(), uint8(s.Final.X), testFile, i, "X") || fail
		fail = !test.ExpectEquality(t, mc.Y.Value(), uint8(s.Final.Y), testFile, i, "Y") || fail
		fail = !test.ExpectEquality(t, mc.SP.Value(), uint8(s.Final.S), testFile, i, "SP") || fail
		fail = !test.ExpectEquality(t, mc.Status.Value()&0xef, uint8(s.Final.P&0xef), testFile, i, "Status") || fail
		for _, r := range s.Final.RAM {
			fail = !test.ExpectEquality(t, mem.internal[r.Address], r.Value,
				testFile, i, fmt.Sprintf("RAM %04x", r.Address)) || fail
		}

		// the number of entries in the cycles array is way too many in the tests. we don't test
		// them because we are confident that the number executed by KIL is correct
		if mc.LastResult.Defn.Operator.String() != "KIL" {
			fail = !test.ExpectEquality(t, mc.LastResult.Cycles, len(s.Cycles), testFile, i, "cycle count") || fail
		}

		if fail {
			fatalTest()
		}
	}
}
