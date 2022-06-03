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

package dpcplus

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/memorymodel"
)

// there is only one version of DPC+ currently but this method of specifying
// addresses mirrors how we do it in the CDF type.
type version struct {
	arch arm.Architecture
	mmap memorymodel.Map

	driverOriginROM uint32
	driverMemtopROM uint32
	customOriginROM uint32
	customMemtopROM uint32
	dataOriginROM   uint32
	dataMemtopROM   uint32
	freqOriginROM   uint32
	freqMemtopROM   uint32
	driverOriginRAM uint32
	driverMemtopRAM uint32
	dataOriginRAM   uint32
	dataMemtopRAM   uint32
	freqOriginRAM   uint32
	freqMemtopRAM   uint32

	// stack should be within the range of the RAM copy of the frequency tables.
	stackOriginRAM uint32
}

func newVersion(memModel string, data []uint8) version {
	var arch arm.Architecture
	var mmap memorymodel.Map

	if memModel == "AUTO" {
		if data[0xc4b]&0x20 == 0x20 && data[0xc4f]&0x20 == 0x20 {
			arch = arm.ARMv7_M
			mmap = memorymodel.NewMap(memorymodel.PlusCart)
		} else {
			arch = arm.ARM7TDMI
			mmap = memorymodel.NewMap(memorymodel.Harmony)
		}
	}

	return version{
		arch:            arch,
		mmap:            mmap,
		driverOriginROM: mmap.FlashOrigin,
		driverMemtopROM: mmap.FlashOrigin | 0x00000bff,
		customOriginROM: mmap.FlashOrigin | 0x00000c00,
		customMemtopROM: mmap.FlashOrigin | 0x00006bff,
		dataOriginROM:   mmap.FlashOrigin | 0x00006c00,
		dataMemtopROM:   mmap.FlashOrigin | 0x00007bff,
		freqOriginROM:   mmap.FlashOrigin | 0x00007c00,
		freqMemtopROM:   mmap.FlashOrigin | 0x00008000,
		driverOriginRAM: mmap.SRAMOrigin | 0x00000000,
		driverMemtopRAM: mmap.SRAMOrigin | 0x00000bff,
		dataOriginRAM:   mmap.SRAMOrigin | 0x00000c00,
		dataMemtopRAM:   mmap.SRAMOrigin | 0x00001bff,
		freqOriginRAM:   mmap.SRAMOrigin | 0x00001c00,
		freqMemtopRAM:   mmap.SRAMOrigin | 0x00002000,

		// stack should be within the range of the RAM copy of the frequency tables.
		stackOriginRAM: mmap.SRAMOrigin | 0x00001fdc,
	}
}
