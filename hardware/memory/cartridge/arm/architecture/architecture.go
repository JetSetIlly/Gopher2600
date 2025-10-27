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

// CartArchitecture defines the memory map for the ARM.
type CartArchitecture string

// List of valid CartArchitecture values.
const (
	Harmony  CartArchitecture = "Harmony"
	PlusCart CartArchitecture = "PlusCart"
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

// Regions of memory in the architecture's map
type MemoryRegion struct {
	Name    string
	Origin  uint32
	Memtop  uint32
	Latency float64

	// whether the memory region is subject to MAM. the architecture might not have MAM at all
	UseMAM bool
}

// Map of the differences between architectures. The differences are led by the
// cartridge architecture.
type Map struct {
	CartArchitecture CartArchitecture
	ARMArchitecture  ARMArchitecture

	// some ARM architectures allow misaligned accesses for some instructions
	MisalignedAccesses bool

	// list of memory regions
	Regions map[string]*MemoryRegion

	// the mask to apply to an address to quickly determine which region it belongs to
	RegionMask uint32

	// returns a numeric region ID for the address. the number of possible IDs is the number of
	// memory regions in the architecture when counted from one. an ID of zero indicates that the
	// address falls outside all the specified regions
	RegionID func(addr uint32) int

	// peripherals

	// MAM
	HasMAM bool
	MAMCR  uint32
	MAMTIM uint32

	// PreferredMAMCR is the value that will be used when ARM MAM preferences
	// is set to driver. defaults to MAMfull and is intended to be altered by
	// the cartridge implementation before creating the ARM emulation
	PreferredMAMCR MAMCR

	HasT1 bool
	T1TCR uint32
	T1TC  uint32

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

	APBDIV uint32

	// the address below which a null access is considered to have happened
	NullAccessBoundary uint32

	// the value the is returned when an illegal memory address is read
	IllegalAccessValue uint32

	// the divisor to apply to the main clock when ticking peripherals (eg. timers)
	ClkDiv float32

	// list of unimplemented address ranges that should be ignored
	Unimplemented []MemoryRegion
}

// NewMap is the preferred method of initialisation for the Map type.
func NewMap(cart CartArchitecture) Map {
	mmap := Map{
		CartArchitecture: cart,
		Regions:          make(map[string]*MemoryRegion),
	}

	switch mmap.CartArchitecture {
	default:
		// logger.Logf(env, "ARM Architecture", "unknown cartridge architecture (%s) defaulting to Harmony", cart)
		mmap.CartArchitecture = Harmony
		fallthrough

	case Harmony:
		mmap.ARMArchitecture = ARM7TDMI
		mmap.MisalignedAccesses = false

		mmap.Regions["Flash"] = &MemoryRegion{
			Name:    "Flash",
			Origin:  0x00000000,
			Memtop:  0x0fffffff,
			Latency: 50.0,
			UseMAM:  true,
		}
		mmap.Regions["SRAM"] = &MemoryRegion{
			Name:    "SRAM",
			Origin:  0x40000000,
			Memtop:  0x4fffffff,
			Latency: 10.0,
		}

		mmap.RegionMask = 0xf0000000

		mmap.RegionID = func(addr uint32) int {
			switch addr & mmap.RegionMask {
			case 0x40000000:
				return 1 // Flash
			case 0x00000000:
				return 2 // SRAM
			}
			return 0
		}

		mmap.HasMAM = true
		mmap.MAMCR = 0xe01fc000
		mmap.MAMTIM = 0xe01fc004
		mmap.PreferredMAMCR = MAMpartial

		mmap.HasT1 = true
		mmap.T1TCR = 0xe0008004
		mmap.T1TC = 0xe0008008

		mmap.APBDIV = 0xe01fc100

		// boundary value is arbitrary and was suggested by John Champeau (09/04/2022)
		mmap.NullAccessBoundary = 0x00000751
		mmap.IllegalAccessValue = 0x00000000

		// from "12. APB Divider" in "UM10161", page 61
		//
		// "Because the APB bus must work properly at power up (and its timing
		// cannot be altered if it does not work since the APB divider control
		// registers reside on the APB bus), the default condition at reset is
		// for the APB bus to run at one quarter speed"
		//
		// in the LPC2000 the ClkDiv value is defined by the APBDIV register.
		// we're not emulating the APBDIV register and assume that the value
		// is set to 0, meaning a PCLK of a quarter of the CCLK (the clock speed
		// of the main processing unit)
		//
		// *** For now, we'll keep this value at clock division of 1 until we
		// understand better what is happening
		mmap.ClkDiv = 1

	case PlusCart:
		mmap.ARMArchitecture = ARMv7_M
		mmap.MisalignedAccesses = true

		mmap.Regions["Flash"] = &MemoryRegion{
			Name:    "Flash",
			Origin:  0x08020000,
			Memtop:  0x0802ffff,
			Latency: 1.0,
		}
		mmap.Regions["SRAM"] = &MemoryRegion{
			Name:    "SRAM",
			Origin:  0x20000000,
			Memtop:  0x2fffffff,
			Latency: 1.0,
		}
		mmap.Regions["CCM"] = &MemoryRegion{
			Name:    "CCM",
			Origin:  0x10000000,
			Memtop:  0x1fffffff,
			Latency: 1.0,
		}

		mmap.RegionMask = 0xf0000000

		mmap.RegionID = func(addr uint32) int {
			switch addr & mmap.RegionMask {
			case 0x00000000:
				return 1 // Flash
			case 0x20000000:
				return 2 // SRAM
			case 0x10000000:
				return 3 // CCM
			}
			return 0
		}

		// there is no MAM in this architecture but the effect of MAMfull is what we want
		mmap.HasMAM = false
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

		mmap.APBDIV = 0x40021004

		// boundary value is arbitrary and was suggested by John Champeau (09/04/2022)
		mmap.NullAccessBoundary = 0x00000751
		mmap.IllegalAccessValue = 0xffffffff

		mmap.ClkDiv = 2

		mmap.Unimplemented = []MemoryRegion{
			{Name: "RCC", Origin: 0x40023800, Memtop: 0x4002388c},
			{Name: "DMA1", Origin: 0x40026000, Memtop: 0x400263ff},
			{Name: "DMA2", Origin: 0x40026400, Memtop: 0x400267ff},
		}
	}

	return mmap
}

func (mmap *Map) IsUnimplemented(addr uint32) (bool, string) {
	for _, u := range mmap.Unimplemented {
		if addr >= u.Origin && addr <= u.Memtop {
			return true, u.Name
		}
	}
	return false, ""
}
