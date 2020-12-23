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
	"fmt"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/logger"
)

// Static implements the bus.CartStatic interface.
type Static struct {
	// copy of entire ROM (for convenience)
	cartDataROM []byte

	// slices of cartDataRAM that will be modified during execution
	driverRAM []byte
	dataRAM   []byte
	freqRAM   []byte

	// slices of cartDataROM that should not be modified during execution
	driverROM []byte
	customROM []byte
	dataROM   []byte
	freqROM   []byte
}

func (cart *dpcPlus) newDPCplusStatic(cartData []byte) *Static {
	mem := Static{
		cartDataROM: cartData,
	}

	// the offset into the cart data where the data segment begins
	dataOffset := driverSize + (cart.bankSize * cart.NumBanks())

	// ARM driver
	mem.driverROM = cartData[:driverSize]

	// custom ARM program immediately after the ARM driver and where we've
	// figured the data segment to start. note that some of this will be the
	// 6507 program but we can't really know for sure where that begins.
	mem.customROM = cartData[driverSize:dataOffset]

	// gfx and frequency table at end of file
	// unlike CDF ROMs data and frequency tables are initialised from the ROM
	mem.dataROM = cartData[dataOffset : dataOffset+dataSize]
	mem.freqROM = cartData[dataOffset+dataSize:]

	// RAM areas
	mem.driverRAM = make([]byte, len(mem.driverROM))
	copy(mem.driverRAM, mem.driverROM)
	mem.dataRAM = make([]byte, len(mem.dataROM))
	copy(mem.dataRAM, mem.dataROM)
	mem.freqRAM = make([]byte, len(mem.freqROM))
	copy(mem.freqRAM, mem.freqROM)

	return &mem
}

// ResetVectors implements the arm7tdmi.SharedMemory interface.
func (mem *Static) ResetVectors() (uint32, uint32, uint32) {
	return stackOriginRAM, customOriginROM, customOriginROM + 8
}

func (mem *Static) Snapshot() *Static {
	n := *mem
	n.driverRAM = make([]byte, len(mem.driverROM))
	copy(n.driverRAM, mem.driverROM)
	n.dataRAM = make([]byte, len(mem.dataROM))
	copy(n.dataRAM, mem.dataROM)
	n.freqRAM = make([]byte, len(mem.freqROM))
	copy(n.freqRAM, mem.freqROM)
	return &n
}

// MapAddress implements the arm7tdmi.SharedMemory interface.
func (mem *Static) MapAddress(addr uint32, write bool) (*[]byte, uint32) {
	// driver ARM code (ROM)
	if addr >= driverOriginROM && addr <= driverMemtopROM {
		if write {
			logger.Log("DPC+", fmt.Sprintf("ARM trying to write to ROM address (%08x)", addr))
			return nil, addr
		}
		return &mem.driverROM, addr - driverOriginROM
	}

	// custom ARM code (ROM)
	if addr >= customOriginROM && addr <= customMemtopROM {
		if write {
			logger.Log("DPC+", fmt.Sprintf("ARM trying to write to ROM address (%08x)", addr))
			return nil, addr
		}
		return &mem.customROM, addr - customOriginROM
	}

	// data (ROM)
	if addr >= dataOriginROM && addr <= dataMemtopROM {
		if write {
			logger.Log("DPC+", fmt.Sprintf("ARM trying to write to ROM address (%08x)", addr))
			return nil, addr
		}
		return &mem.dataROM, addr - dataOriginROM
	}

	// frequency table (ROM)
	if addr >= freqOriginROM && addr <= freqMemtopROM {
		if write {
			logger.Log("DPC+", fmt.Sprintf("ARM trying to write to ROM address (%08x)", addr))
			return nil, addr
		}
		return &mem.freqROM, addr - freqOriginROM
	}

	// driver ARM code (RAM)
	if addr >= driverOriginRAM && addr <= driverMemtopRAM {
		return &mem.driverRAM, addr - driverOriginRAM
	}

	// data (RAM)
	if addr >= dataOriginRAM && addr <= dataMemtopRAM {
		return &mem.dataRAM, addr - dataOriginRAM
	}

	// frequency table (RAM)
	if addr >= freqOriginRAM && addr <= freqMemtopRAM {
		return &mem.freqRAM, addr - freqOriginRAM
	}

	return nil, addr
}

// GetStatic implements the bus.CartDebugBus interface.
func (cart *dpcPlus) GetStatic() []mapper.CartStatic {
	s := make([]mapper.CartStatic, 3)

	s[0].Segment = "Driver"
	s[1].Segment = "Data"
	s[2].Segment = "Freq"

	s[0].Data = make([]byte, len(cart.state.static.driverRAM))
	s[1].Data = make([]byte, len(cart.state.static.dataRAM))
	s[2].Data = make([]byte, len(cart.state.static.freqRAM))

	copy(s[0].Data, cart.state.static.driverRAM)
	copy(s[1].Data, cart.state.static.dataRAM)
	copy(s[1].Data, cart.state.static.freqRAM)

	return s
}

// StaticWrite implements the bus.CartDebugBus interface.
func (cart *dpcPlus) PutStatic(segment string, idx uint16, data uint8) error {
	switch segment {
	case "Driver":
		if int(idx) >= len(cart.state.static.driverRAM) {
			return curated.Errorf("CDFJ", fmt.Errorf("index too high (%#04x) for %s area", idx, segment))
		}
		cart.state.static.driverRAM[idx] = data

	case "Data":
		if int(idx) >= len(cart.state.static.dataRAM) {
			return curated.Errorf("DPC+: static: %v", fmt.Errorf("index too high (%#04x) for %s area", idx, segment))
		}
		cart.state.static.dataRAM[idx] = data

	case "Freq":
		if int(idx) >= len(cart.state.static.freqRAM) {
			return curated.Errorf("DPC+: static: %v", fmt.Errorf("index too high (%#04x) for %s area", idx, segment))
		}
		cart.state.static.freqRAM[idx] = data

	default:
		return curated.Errorf("DPC+: static: %v", fmt.Errorf("unknown segment (%s)", segment))
	}

	return nil
}
