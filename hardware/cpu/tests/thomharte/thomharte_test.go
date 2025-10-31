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
	"os"
	"path/filepath"
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

func (mem testMem) ReadZeroPage(address uint8) (uint8, error) {
	return mem.Read(uint16(address))
}

func (mem *testMem) Write(address uint16, data uint8) error {
	mem.internal[address] = data
	return nil
}

type ramState [2]uint16

type State struct {
	PC  uint64     `json:"pc"`
	S   uint64     `json:"s"`
	A   uint64     `json:"a"`
	X   uint64     `json:"x"`
	Y   uint64     `json:"y"`
	P   uint64     `json:"p"`
	RAM []ramState `json:"ram"`
}

type Tests struct {
	Name    string `json:"name"`
	Initial State  `json:"initial"`
	Final   State  `json:"final"`
	// ignoring 'cycles' entries. we're not testing that
}

var testsPath = filepath.Join("6502", "v1")

func TestThomHarte(t *testing.T) {
	d, err := os.ReadDir(testsPath)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range d {
		switch e.Name() {
		case ".gitkeep":
			continue
		}
		if e.Type().IsRegular() {
			testThomHarte(t, filepath.Join(testsPath, e.Name()))
		}
	}
}

func testThomHarte(t *testing.T, testFile string) {
	t.Logf("testing %s", testFile)

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
	mc.Reset(nil)

	for i, s := range tests {
		for _, r := range s.Initial.RAM {
			mem.internal[r[0]] = uint8(r[1])
		}
		mc.PC.Load(uint16(s.Initial.PC))
		mc.A.Load(uint8(s.Initial.A))
		mc.X.Load(uint8(s.Initial.X))
		mc.Y.Load(uint8(s.Initial.Y))
		mc.SP.Load(uint8(s.Initial.S))
		mc.Status.Load(uint8(s.Initial.P))

		err := mc.ExecuteInstruction(cpu.NilCycleCallback)
		if err != nil {
			t.Fatal(err)
		}

		var fail bool

		fail = !test.ExpectEquality(t, mc.PC.Value(), uint16(s.Final.PC), testFile, i, "PC") || fail
		fail = !test.ExpectEquality(t, mc.A.Value(), uint8(s.Final.A), testFile, i, "A") || fail
		fail = !test.ExpectEquality(t, mc.X.Value(), uint8(s.Final.X), testFile, i, "X") || fail
		fail = !test.ExpectEquality(t, mc.Y.Value(), uint8(s.Final.Y), testFile, i, "Y") || fail
		fail = !test.ExpectEquality(t, mc.SP.Value(), uint8(s.Final.S), testFile, i, "SP") || fail
		fail = !test.ExpectEquality(t, mc.Status.Value()&0xef, uint8(s.Final.P), testFile, i, "Status") || fail

		if fail {
			t.Fatalf("%s: failed on line %d", testFile, i)
		}
	}
}
