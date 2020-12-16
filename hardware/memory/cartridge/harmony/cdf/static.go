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

package cdf

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/logger"
)

// Static implements the bus.CartStatic interface.
type Static struct {
	cartDataROM []byte
	cartDataRAM []byte

	// slices of cartDataROM that should not be modified during execution
	driverROM []byte
	customROM []byte

	// slices of cartData that will be modified during execution
	driverRAM    []byte
	dataRAM      []byte
	variablesRAM []byte
}

func (cart *cdf) newCDFstatic(cartData []byte) *Static {
	stc := Static{
		cartDataRAM: cartData,
	}

	// make a copy for non-volatile purposes
	stc.cartDataROM = make([]byte, len(cartData))
	copy(stc.cartDataROM, cartData)

	// ARM driver
	stc.driverROM = stc.cartDataROM[:driverSize]
	stc.driverRAM = stc.cartDataRAM[:driverSize]

	// custom ARM program begins immediately after the ARM driver
	stc.customROM = stc.cartDataROM[driverSize:]

	// variables nothing in the ROM data we can use (unlike DPC+) so we must
	// allocate fresh memory
	stc.dataRAM = make([]byte, dataMemtopRAM-dataOriginRAM+1)
	stc.variablesRAM = make([]byte, variablesMemtopRAM-variablesOriginRAM+1)

	return &stc
}

// ResetVectors implements the arm7tdmi.SharedMemory interface.
func (stc *Static) ResetVectors() (uint32, uint32, uint32) {
	return stackOriginRAM, customOriginROM, customOriginROM + 8
}

// the memory addresses from the point of view of the ARM processor.
const (
	driverOriginROM = 0x00000000
	driverMemtopROM = 0x000007ff

	customOriginROM = 0x00000800
	customMemtopROM = 0x00007fff

	driverOriginRAM = 0x40000000
	driverMemtopRAM = 0x400007ff

	dataOriginRAM = 0x40000800
	dataMemtopRAM = 0x400017ff

	variablesOriginRAM = 0x40001800
	variablesMemtopRAM = 0x40001fff

	// stack should be within the range of the RAM copy of the variables
	stackOriginRAM = 0x40001fdc
)

// MapAddress implements the arm7tdmi.SharedMemory interface.
func (stc *Static) MapAddress(addr uint32, write bool) (*[]byte, uint32) {
	// driver ARM code (ROM)
	if addr >= driverOriginROM && addr <= driverMemtopROM {
		return &stc.driverROM, addr - driverOriginROM
	}

	// custom ARM code (ROM)
	if addr >= customOriginROM && addr <= customMemtopROM {
		if write {
			logger.Log("CDF", fmt.Sprintf("ARM trying to write to ROM address (%08x)", addr))
			return nil, addr
		}
		return &stc.customROM, addr - customOriginROM
	}

	// driver ARM code (RAM)
	if addr >= driverOriginRAM && addr <= driverMemtopRAM {
		return &stc.driverRAM, addr - driverOriginRAM
	}

	// data (RAM)
	if addr >= dataOriginRAM && addr <= dataMemtopRAM {
		return &stc.dataRAM, addr - dataOriginRAM
	}

	// variables (RAM)
	if addr >= variablesOriginRAM && addr <= variablesMemtopRAM {
		return &stc.variablesRAM, addr - variablesOriginRAM
	}

	return nil, addr
}

func (stc *Static) read8bit(addr uint32) uint8 {
	mem, addr := stc.MapAddress(addr, false)
	if mem == nil || addr >= uint32(len(*mem)) {
		return 0
	}
	return (*mem)[addr]
}

func (stc *Static) read16bit(addr uint32) uint16 {
	mem, addr := stc.MapAddress(addr, false)
	if mem == nil || addr >= uint32(len(*mem)) {
		return 0
	}
	return uint16((*mem)[addr]) |
		uint16((*mem)[addr+1])<<8
}

func (stc *Static) read32bit(addr uint32) uint32 {
	mem, addr := stc.MapAddress(addr, false)
	if mem == nil || addr >= uint32(len(*mem)) {
		return 0
	}
	return uint32((*mem)[addr]) |
		uint32((*mem)[addr+1])<<8 |
		uint32((*mem)[addr+2])<<16 |
		uint32((*mem)[addr+3])<<24
}

// GetStatic implements the bus.CartDebugBus interface.
func (cart *cdf) GetStatic() []mapper.CartStatic {
	s := make([]mapper.CartStatic, 3)

	s[0].Segment = "Driver"
	s[1].Segment = "Data"
	s[2].Segment = "Variables"

	s[0].Data = make([]byte, len(cart.static.driverRAM))
	s[1].Data = make([]byte, len(cart.static.dataRAM))
	s[2].Data = make([]byte, len(cart.static.variablesRAM))

	copy(s[0].Data, cart.static.driverRAM)
	copy(s[1].Data, cart.static.dataRAM)
	copy(s[2].Data, cart.static.variablesRAM)

	return s
}

// StaticWrite implements the bus.CartDebugBus interface.
func (cart *cdf) PutStatic(segment string, idx uint16, data uint8) error {
	switch segment {
	case "Driver":
		if int(idx) >= len(cart.static.driverRAM) {
			return curated.Errorf("CDF", fmt.Errorf("index too high (%#04x) for %s area", idx, segment))
		}
		cart.static.driverRAM[idx] = data

	case "Data":
		if int(idx) >= len(cart.static.dataRAM) {
			return curated.Errorf("CDF", fmt.Errorf("index too high (%#04x) for %s area", idx, segment))
		}
		cart.static.dataRAM[idx] = data

	case "Variables":
		if int(idx) >= len(cart.static.variablesRAM) {
			return curated.Errorf("CDF", fmt.Errorf("index too high (%#04x) for %s area", idx, segment))
		}
		cart.static.variablesRAM[idx] = data

	default:
		return curated.Errorf("CDF", fmt.Errorf("unknown segment (%s)", segment))
	}

	return nil
}
