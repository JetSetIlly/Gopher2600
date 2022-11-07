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

// Package architecture defines the Map type that is used to specify the
// differences in cartridge and ARM archtectures.
package architecture

import (
	"github.com/jetsetilly/gopher2600/logger"
)

// CartArchitecture defines the memory map for the ARM.
type CartArchitecture string

// List of valid CartArchitecture values.
const (
	Harmony  CartArchitecture = "LPC2000"
	PlusCart CartArchitecture = "STM32F407VGT6"
)

// ARMArchitecture defines the features of the ARM core.
type ARMArchitecture string

// List of valid ARMArchitecture values.
const (
	ARM7TDMI ARMArchitecture = "ARM7TDMI"
	ARMv7_M  ARMArchitecture = "ARMv7-M"
)

// MAMCR defines the state of the MAM.
type MAMCR uint32

// List of valid MAMCR values.
const (
	MAMdisabled MAMCR = iota
	MAMpartial
	MAMfull
)

// Map of the differences between architectures. The differences are led by the
// cartridge architecture.
type Map struct {
	CartArchitecture CartArchitecture
	ARMArchitecture  ARMArchitecture

	FlashOrigin    uint32
	Flash32kMemtop uint32
	Flash64kMemtop uint32
	FlashMaxMemtop uint32
	SRAMOrigin     uint32

	// the memory latency of the Flash memory block (in nanoseconds)
	FlashLatency float64

	// peripherals

	// MAM
	HasMAM bool
	MAMCR  uint32
	MAMTIM uint32

	// PreferredMAMCR is the value that will be used when ARM MAM preferences
	// is set to driver. defaults to MAMfull and is intended to be altered by
	// the cartridge implementation before creating the ARM emulation
	PreferredMAMCR MAMCR

	HasTIMER     bool
	TIMERcontrol uint32
	TIMERvalue   uint32

	HasTIM2 bool
	TIM2CR1 uint32
	TIM2EGR uint32
	TIM2CNT uint32
	TIM2PSC uint32
	TIM2ARR uint32

	HasRNG bool
	RNGCR  uint32
	RNGSR  uint32
	RNGDR  uint32

	// the address below which a null access is considered to have happened
	NullAccessBoundary uint32

	// the value the is returned when an illegal memory address is read
	IllegalAccessValue uint32

	// the divisor to apply to the main clock when ticking the timers
	ClkDiv float32

	// whether to the processor to trap (or log) a disallowed unaligned memory access
	UnalignTrap bool
}

// NewMap is the preferred method of initialisation for the Map type.
func NewMap(cart CartArchitecture) Map {
	mmap := Map{
		CartArchitecture: cart,
	}

	switch mmap.CartArchitecture {
	default:
		logger.Logf("ARM Architecture", "unknown cartridge architecture (%s) defaulting to Harmony", cart)
		mmap.CartArchitecture = Harmony
		fallthrough

	case Harmony:
		mmap.ARMArchitecture = ARM7TDMI

		mmap.FlashOrigin = 0x00000000
		mmap.Flash32kMemtop = 0x00007fff
		mmap.Flash64kMemtop = 0x000fffff
		mmap.FlashMaxMemtop = 0x0fffffff
		mmap.SRAMOrigin = 0x40000000

		mmap.FlashLatency = 50.0

		mmap.MAMCR = 0x001fc000
		mmap.MAMTIM = 0x001fc004
		mmap.PreferredMAMCR = MAMpartial

		mmap.HasTIMER = true
		mmap.TIMERcontrol = 0xe0008004
		mmap.TIMERvalue = 0xe0008008

		// boundary value is arbitrary and was suggested by John Champeau (09/04/2022)
		mmap.NullAccessBoundary = 0x00000751
		mmap.IllegalAccessValue = 0x00000000

		mmap.ClkDiv = 1.0
		mmap.UnalignTrap = true

	case PlusCart:
		mmap.ARMArchitecture = ARMv7_M

		mmap.FlashOrigin = 0x20000000
		mmap.Flash32kMemtop = 0x20007fff
		mmap.Flash64kMemtop = 0x200fffff
		mmap.FlashMaxMemtop = 0x2fffffff
		mmap.SRAMOrigin = 0x10000000

		mmap.FlashLatency = 10.0

		mmap.PreferredMAMCR = MAMfull

		mmap.HasTIM2 = true
		mmap.TIM2CR1 = 0x40000000
		mmap.TIM2EGR = 0x40000014
		mmap.TIM2CNT = 0x40000024
		mmap.TIM2PSC = 0x40000028
		mmap.TIM2ARR = 0x4000002c

		mmap.HasRNG = true
		mmap.RNGCR = 0x50060800
		mmap.RNGSR = 0x50060804
		mmap.RNGDR = 0x50060808

		// boundary value is arbitrary and was suggested by John Champeau (09/04/2022)
		mmap.NullAccessBoundary = 0x00000751
		mmap.IllegalAccessValue = 0xffffffff

		mmap.ClkDiv = 0.5
		mmap.UnalignTrap = false
	}

	logger.Logf("ARM Architecture", "using %s/%s", mmap.CartArchitecture, mmap.ARMArchitecture)
	logger.Logf("ARM Architecture", "flash origin: %#08x", mmap.FlashOrigin)
	logger.Logf("ARM Architecture", "sram origin: %#08x", mmap.SRAMOrigin)

	return mmap
}

// IsFlash returns true if address is in flash memory range.
func (mmap *Map) IsFlash(addr uint32) bool {
	return addr >= mmap.FlashOrigin && addr <= mmap.Flash64kMemtop
}
