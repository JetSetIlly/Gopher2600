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
	version version

	// slices of cartData that should not be modified during execution
	driverROM []byte
	customROM []byte

	// slices of cartData that will be modified during execution
	driverRAM    []byte
	dataRAM      []byte
	variablesRAM []byte
}

func (cart *cdf) newCDFstatic(cartData []byte, r version) *Static {
	stc := Static{version: r}

	// ARM driver
	stc.driverROM = cartData[:driverSize]

	// custom ARM program begins immediately after the ARM driver
	stc.customROM = cartData[r.customOriginROM-stc.version.mmap.FlashOrigin:]

	// driver RAM is the same as driver ROM initially
	stc.driverRAM = make([]byte, driverSize)
	copy(stc.driverRAM, stc.driverROM)

	// there is nothing in cartData to copy into the other RAM areas
	stc.dataRAM = make([]byte, r.dataMemtopRAM-r.dataOriginRAM+1)
	stc.variablesRAM = make([]byte, r.variablesMemtopRAM-r.variablesOriginRAM+1)

	return &stc
}

func (stc *Static) HotLoad(cartData []byte) {
	stc.driverROM = cartData[:driverSize]
	stc.customROM = cartData[driverSize:]
}

// ResetVectors implements the arm7tdmi.SharedMemory interface.
func (stc *Static) ResetVectors() (uint32, uint32, uint32) {
	return stc.version.entrySR, stc.version.entryLR, stc.version.entryPC
}

func (stc *Static) Snapshot() *Static {
	n := *stc
	n.driverRAM = make([]byte, len(stc.driverRAM))
	n.dataRAM = make([]byte, len(stc.dataRAM))
	n.variablesRAM = make([]byte, len(stc.variablesRAM))
	copy(n.driverRAM, stc.driverRAM)
	copy(n.dataRAM, stc.dataRAM)
	copy(n.variablesRAM, stc.variablesRAM)
	return &n
}

// MapAddress implements the arm7tdmi.SharedMemory interface.
func (stc *Static) MapAddress(addr uint32, write bool) (*[]byte, uint32) {
	// tests arranged in order of most likely to be used

	// custom ARM code (ROM)
	if addr >= stc.version.customOriginROM && addr <= stc.version.customMemtopROM {
		if write {
			logger.Logf("CDF", "ARM trying to write to ROM address (%08x)", addr)
			return nil, addr
		}
		return &stc.customROM, addr - stc.version.customOriginROM
	}

	// driver ARM code (ROM)
	if addr >= stc.version.driverOriginROM && addr <= stc.version.driverMemtopROM {
		return &stc.driverROM, addr - stc.version.driverOriginROM
	}

	// data (RAM)
	if addr >= stc.version.dataOriginRAM && addr <= stc.version.dataMemtopRAM {
		return &stc.dataRAM, addr - stc.version.dataOriginRAM
	}

	// driver ARM code (RAM)
	if addr >= stc.version.driverOriginRAM && addr <= stc.version.driverMemtopRAM {
		return &stc.driverRAM, addr - stc.version.driverOriginRAM
	}

	// variables (RAM)
	if addr >= stc.version.variablesOriginRAM && addr <= stc.version.variablesMemtopRAM {
		return &stc.variablesRAM, addr - stc.version.variablesOriginRAM
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

// func (stc *Static) read16bit(addr uint32) uint16 {
// 	mem, addr := stc.MapAddress(addr, false)
// 	if mem == nil || addr >= uint32(len(*mem)) {
// 		return 0
// 	}
// 	return uint16((*mem)[addr]) |
// 		uint16((*mem)[addr+1])<<8
// }.

// func (stc *Static) read32bit(addr uint32) uint32 {
// 	mem, addr := stc.MapAddress(addr, false)
// 	if mem == nil || addr >= uint32(len(*mem)) {
// 		return 0
// 	}
// 	return uint32((*mem)[addr]) |
// 		uint32((*mem)[addr+1])<<8 |
// 		uint32((*mem)[addr+2])<<16 |
// 		uint32((*mem)[addr+3])<<24
// }.

// GetStatic implements the bus.CartDebugBus interface.
func (cart *cdf) GetStatic() []mapper.CartStatic {
	numSegments := 3
	if cart.version.submapping == "CDFJ+" {
		numSegments = 2
	}

	s := make([]mapper.CartStatic, numSegments)

	s[0].Segment = "Driver"
	s[1].Segment = "Data"

	s[0].Data = make([]byte, len(cart.state.static.driverRAM))
	s[1].Data = make([]byte, len(cart.state.static.dataRAM))

	copy(s[0].Data, cart.state.static.driverRAM)
	copy(s[1].Data, cart.state.static.dataRAM)

	if cart.version.submapping != "CDFJ+" {
		s[2].Segment = "Variables"
		s[2].Data = make([]byte, len(cart.state.static.variablesRAM))
		copy(s[2].Data, cart.state.static.variablesRAM)
	}

	return s
}

// StaticWrite implements the bus.CartDebugBus interface.
func (cart *cdf) PutStatic(segment string, idx uint16, data uint8) error {
	switch segment {
	case "Driver":
		if int(idx) >= len(cart.state.static.driverRAM) {
			return curated.Errorf("CDF", fmt.Errorf("index too high (%#04x) for %s area", idx, segment))
		}
		cart.state.static.driverRAM[idx] = data

	case "Data":
		if int(idx) >= len(cart.state.static.dataRAM) {
			return curated.Errorf("CDF", fmt.Errorf("index too high (%#04x) for %s area", idx, segment))
		}
		cart.state.static.dataRAM[idx] = data

	case "Variables":
		if int(idx) >= len(cart.state.static.variablesRAM) {
			return curated.Errorf("CDF", fmt.Errorf("index too high (%#04x) for %s area", idx, segment))
		}
		cart.state.static.variablesRAM[idx] = data

	default:
		return curated.Errorf("CDF", fmt.Errorf("unknown segment (%s)", segment))
	}

	return nil
}
