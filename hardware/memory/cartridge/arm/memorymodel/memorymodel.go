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

// Package memorymodel handles differences in memory addressing For example,
// the Harmony family is different to the PlusCart family.
package memorymodel

import (
	"github.com/jetsetilly/gopher2600/logger"
)

type Map struct {
	Model string

	FlashOrigin    uint32
	Flash32kMemtop uint32
	Flash64kMemtop uint32
	FlashMaxMemtop uint32
	SRAMOrigin     uint32

	// peripherals

	// MAM
	HasMAM bool
	MAMCR  uint32
	MAMTIM uint32

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

	// the divisor to apply to the main clock when ticking the timers
	ClkDiv float32

	// whether to the processor to trap (or log) a disallowed unaligned memory access
	UnalignTrap bool
}

const (
	Harmony  = "LPC2000"
	PlusCart = "STM32F407VGT6"
)

// NewMap is the preferred method of initialisation for the Map type.
func NewMap(model string) Map {
	mmap := Map{
		Model: model,
	}

	switch mmap.Model {
	default:
		logger.Logf("ARM Mem Model", "unknown ARM memory model (%s) defaulting to Harmony", mmap.Model)
		fallthrough

	case Harmony:
		mmap.FlashOrigin = 0x00000000
		mmap.Flash32kMemtop = 0x00007fff
		mmap.Flash64kMemtop = 0x000fffff
		mmap.FlashMaxMemtop = 0x0fffffff
		mmap.SRAMOrigin = 0x40000000

		mmap.MAMCR = 0x001fc000
		mmap.MAMTIM = 0x001fc004

		mmap.HasTIMER = true
		mmap.TIMERcontrol = 0xe0008004
		mmap.TIMERvalue = 0xe0008008

		// value is arbitrary and was suggested by John Champeau (09/04/2022)
		mmap.NullAccessBoundary = 0x00000751

		mmap.ClkDiv = 1.0
		mmap.UnalignTrap = true

	case PlusCart:
		mmap.FlashOrigin = 0x20000000
		mmap.Flash32kMemtop = 0x20007fff
		mmap.Flash64kMemtop = 0x200fffff
		mmap.FlashMaxMemtop = 0x2fffffff
		mmap.SRAMOrigin = 0x10000000

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

		mmap.ClkDiv = 0.5
		mmap.UnalignTrap = false
	}

	logger.Logf("ARM Mem Model", "using %s", mmap.Model)
	logger.Logf("ARM Mem Model", "flash origin: %#08x", mmap.FlashOrigin)
	logger.Logf("ARM Mem Model", "sram origin: %#08x", mmap.SRAMOrigin)

	return mmap
}

// IsFlash returns true if address is in flash memory range.
func (mmap Map) IsFlash(addr uint32) bool {
	return addr >= mmap.FlashOrigin && addr <= mmap.Flash64kMemtop
}
