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

func readDataZeroPage(t *testing.T, mem *memory.Memory, address uint8, expectedData uint8) {
	t.Helper()
	d, err := mem.ReadZeroPage(address)
	if err != nil {
		t.Errorf("unexpected error (%s)", err)
	}
	if d != expectedData {
		t.Errorf("expecting %#02x received %#02x", expectedData, d)
	}
}

func TestDataMask(t *testing.T) {
	mem := memory.NewMemory(nil)

	// no data in register
	readDataZeroPage(t, mem, 0x00, 0x00)
	readDataZeroPage(t, mem, 0x02, 0x02)
	readData(t, mem, 0x02, 0x00)
	readData(t, mem, 0x171, 0x01)

	// high bits set in register
	mem.TIA.ChipWrite(0x00, 0x40)
	readData(t, mem, 0x00, 0x40)
	mem.TIA.ChipWrite(0x01, 0x80)
	readDataZeroPage(t, mem, 0x01, 0x81)

	// low bits set too. low bits in address supercede low bits in register
	mem.TIA.ChipWrite(0x01, 0x8f)
	readDataZeroPage(t, mem, 0x01, 0x81)
	readData(t, mem, 0x01, 0x80)

	mem.TIA.ChipWrite(0x02, 0xc7)
	readDataZeroPage(t, mem, 0x02, 0xc2)
	readData(t, mem, 0x02, 0xc0)

	// non-zero-page addressing
	readData(t, mem, 0x171, 0x81)
}
