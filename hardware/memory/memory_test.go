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

package memory_test

import (
	"testing"

	"github.com/jetsetilly/gopher2600/hardware/memory"
)

func readData(t *testing.T, mem *memory.Memory, address uint16, expectedData uint8) {
	t.Helper()
	d, err := mem.Read(address)
	if err != nil {
		t.Errorf("unexpected error (%s)", err)
	}
	if d != expectedData {
		t.Errorf("expecting %#02x received %#02x", expectedData, d)
	}
}

func writeDataNotTested(t *testing.T, mem *memory.Memory, address uint16, value uint8) {
	err := mem.Write(address, value)
	if err != nil {
		t.Errorf("unexpected error (%s)", err)
	}
}

func readDataNotTested(t *testing.T, mem *memory.Memory, address uint16) {
	_, err := mem.Read(address)
	if err != nil {
		t.Errorf("unexpected error (%s)", err)
	}
}

func TestTIADrivenPins(t *testing.T) {
	mem := memory.NewMemory(nil)

	// preare some test memory
	writeDataNotTested(t, mem, 0x80, 0xff)
	writeDataNotTested(t, mem, 0x81, 0x55)
	writeDataNotTested(t, mem, 0x82, 0xfe)

	readDataNotTested(t, mem, 0x80)
	readData(t, mem, 0x02, 0x3f)

	readDataNotTested(t, mem, 0x81)
	readData(t, mem, 0x02, 0x15)

	// non-zero-page addressing
	readDataNotTested(t, mem, 0x82)
	readData(t, mem, 0x171, 0x3e)
}

func TestAddressComplete(t *testing.T) {
	// this is a very simple test to make sure the memory system is okay with
	// every address. we're not interested in results and we don't expect any
	// errors

	mem := memory.NewMemory(nil)

	for a := 0; a <= 0xffff; a++ {
		_, err := mem.Read(uint16(a))
		if err != nil {
			t.Fail()
		}
	}

	for a := 0; a <= 0xffff; a++ {
		err := mem.Write(uint16(a), 0)
		if err != nil {
			t.Fail()
		}
	}
}
