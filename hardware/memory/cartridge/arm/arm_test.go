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

package arm_test

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/memorymodel"
	"github.com/jetsetilly/gopher2600/hardware/preferences"
)

type testMemory struct {
	data []byte
}

func prepareTestMemory(size uint32) *testMemory {
	return &testMemory{
		data: make([]byte, size),
	}
}

func (mem *testMemory) MapAddress(addr uint32, write bool) (*[]byte, uint32) {
	return &mem.data, 0
}

func (mem *testMemory) ResetVectors() (uint32, uint32, uint32) {
	return 0x3ff, 0x0, 0x0
}

func (mem *testMemory) SoftwareInterrupt(offset int) {
	mem.data[offset+1] = 0xbe
	mem.data[offset] = 0x00
}

type hook struct{}

func (_ *hook) ARMinterrupt(addr uint32, val1 uint32, val2 uint32) (arm.ARMinterruptReturn, error) {
	return arm.ARMinterruptReturn{}, nil
}

func prepareTestARM() (*arm.ARM, *testMemory) {

	prefs := &preferences.ARMPreferences{}

	memModel := memorymodel.Map{
		Model:             "test",
		FlashOrigin:       0,
		Flash32kMemtop:    511,
		Flash64kMemtop:    511,
		SRAMOrigin:        512,
		PeripheralsOrigin: 1000,
		PeripheralsMemtop: 1023,
	}

	testMem := prepareTestMemory(memModel.PeripheralsMemtop)
	hook := &hook{}
	return arm.NewARM(arm.ARMv7_M, arm.MAMdisabled, memModel, prefs, testMem, hook), testMem
}
