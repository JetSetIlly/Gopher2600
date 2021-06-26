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

package arm7tdmi

import (
	"github.com/jetsetilly/gopher2600/logger"
)

type MemoryMap struct {
	Model string

	FlashOrigin       uint32
	Flash32kMemtop    uint32
	Flash64kMemtop    uint32
	SRAMOrigin        uint32
	PeripheralsOrigin uint32
	PeripheralsMemtop uint32

	// specific registers addresses
	TIMERcontrol uint32
	TIMERvalue   uint32
	MAMCR        uint32
	MAMTIM       uint32
}

// NewMemoryMap is the preferred method of initialisation for the MemoryMap type.
func NewMemoryMap(model string) MemoryMap {
	mmap := MemoryMap{
		Model: model,
	}

	switch mmap.Model {
	default:
		logger.Logf("ARM7", "unknown ARM7 model (%s). defaulting to LPC2000", mmap.Model)
		fallthrough

	case "LPC2000":
		// Harmony
		mmap.FlashOrigin = uint32(0x00000000)
		mmap.Flash32kMemtop = uint32(0x00007fff)
		mmap.Flash64kMemtop = uint32(0x000fffff)
		mmap.SRAMOrigin = uint32(0x40000000)
		mmap.PeripheralsOrigin = uint32(0xe0000000)
		mmap.PeripheralsMemtop = uint32(0xffffffff)
		mmap.TIMERcontrol = mmap.PeripheralsOrigin | 0x00008004
		mmap.TIMERvalue = mmap.PeripheralsOrigin | 0x00008008
		mmap.MAMCR = mmap.PeripheralsOrigin | 0x001fc000
		mmap.MAMTIM = mmap.PeripheralsOrigin | 0x001fc004

	case "STM32F407VGT6":
		// PlusCart/UnoCart
		mmap.FlashOrigin = uint32(0x20000000)
		mmap.Flash32kMemtop = uint32(0x20007fff)
		mmap.Flash64kMemtop = uint32(0x200fffff)
		mmap.SRAMOrigin = uint32(0x10000000)
		mmap.PeripheralsOrigin = uint32(0xe0000000)
		mmap.PeripheralsMemtop = uint32(0xffffffff)
	}

	logger.Logf("ARM7", "using %s", mmap.Model)
	logger.Logf("ARM7", "flash origin: %#08x", mmap.FlashOrigin)
	logger.Logf("ARM7", "sram origin: %#08x", mmap.SRAMOrigin)

	return mmap
}

func (mmap MemoryMap) isFlash(addr uint32) bool {
	return addr >= mmap.FlashOrigin && addr <= mmap.Flash64kMemtop
}
