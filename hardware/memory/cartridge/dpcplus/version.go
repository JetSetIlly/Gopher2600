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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/architecture"
)

// there is only one version of DPC+ currently but this method of specifying
// addresses mirrors how we do it in the CDF type.
type version struct {
	mmap architecture.Map

	// segment origins
	driverROMOrigin uint32
	customROMOrigin uint32
	dataROMOrigin   uint32
	freqROMOrigin   uint32
	driverRAMOrigin uint32
	dataRAMOrigin   uint32
	freqRAMOrigin   uint32

	// the memtop values in the version struct are the absolute maximum size
	// supported by the format. the actual memtop may be different depending on
	// the cartridge. the real memtop for a segment should not exceed these
	// maximum values
	driverROMMemtop uint32
	customROMMemtop uint32
	dataROMMemtop   uint32
	freqROMMemtop   uint32
	driverRAMMemtop uint32
	dataRAMMemtop   uint32
	freqRAMMemtop   uint32

	// stack should be within the range of the RAM copy of the frequency tables.
	stackOrigin uint32
}

func newVersion(memModel string, data []uint8) (version, error) {
	var mmap architecture.Map

	switch memModel {
	case "AUTO":
		mmap = architecture.NewMap(architecture.Harmony)

	case "LPC2000":
		// older preference value. deprecated
		fallthrough
	case "ARM7TDMI":
		// old value used to indicate ARM7TDMI architecture. easiest to support
		// it here in this manner
		mmap = architecture.NewMap(architecture.Harmony)

	case "STM32F407VGT6":
		// older preference value. deprecated
		fallthrough
	case "ARMv7_M":
		// old value used to indicate ARM7TDMI architecture. easiest to support
		// it here in this manner
		mmap = architecture.NewMap(architecture.PlusCart)
	}

	return version{
		mmap:            mmap,
		driverROMOrigin: mmap.FlashOrigin,
		driverROMMemtop: mmap.FlashOrigin | 0x00000bff,
		customROMOrigin: mmap.FlashOrigin | 0x00000c00,
		customROMMemtop: mmap.FlashOrigin | 0x00006bff,
		dataROMOrigin:   mmap.FlashOrigin | 0x00006c00,
		dataROMMemtop:   mmap.FlashOrigin | 0x00007bff,
		freqROMOrigin:   mmap.FlashOrigin | 0x00007c00,
		freqROMMemtop:   mmap.FlashOrigin | 0x00008000,
		driverRAMOrigin: mmap.SRAMOrigin | 0x00000000,
		driverRAMMemtop: mmap.SRAMOrigin | 0x00000bff,
		dataRAMOrigin:   mmap.SRAMOrigin | 0x00000c00,
		dataRAMMemtop:   mmap.SRAMOrigin | 0x00001bff,
		freqRAMOrigin:   mmap.SRAMOrigin | 0x00001c00,
		freqRAMMemtop:   mmap.SRAMOrigin | 0x00002000,

		// stack should be within the range of the RAM copy of the frequency tables.
		stackOrigin: mmap.SRAMOrigin | 0x00001fdc,
	}, nil
}
