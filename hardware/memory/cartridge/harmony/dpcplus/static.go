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

// DPCplusStatic implements the bus.CartStatic interface.
type DPCplusStatic struct {
	// full copies of the entire cartridge
	cartDataRAM []byte
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

func (cart *dpcPlus) newDPCplusStatic(cartData []byte) *DPCplusStatic {
	mem := DPCplusStatic{
		cartDataRAM: cartData,
	}

	// make a copy for non-volatile purposes
	mem.cartDataROM = make([]byte, len(cartData))
	copy(mem.cartDataROM, cartData)

	// the offset into the cart data where the data segment begins
	dataOffset := driverSize + (cart.bankSize * cart.NumBanks())

	// ARM driver
	mem.driverRAM = mem.cartDataRAM[:driverSize]
	mem.driverROM = mem.cartDataROM[:driverSize]

	// custom ARM program immediately after the ARM driver and where we've
	// figured the data segment to start. note that some of this will be the
	// 6507 program but we can't really know for sure where that begins.
	mem.customROM = mem.cartDataRAM[driverSize:dataOffset]

	// gfx and frequency table at end of file
	mem.dataRAM = mem.cartDataRAM[dataOffset : dataOffset+dataSize]
	mem.freqRAM = mem.cartDataRAM[dataOffset+dataSize:]
	mem.dataROM = mem.cartDataROM[dataOffset : dataOffset+dataSize]
	mem.freqROM = mem.cartDataROM[dataOffset+dataSize:]

	return &mem
}

// ResetVectors implements the arm7tdmi.SharedMemory interface.
func (mem *DPCplusStatic) ResetVectors() (uint32, uint32, uint32) {
	return stackOriginRAM, customOriginROM, customOriginROM + 8
}

// the memory addresses from the point of view of the ARM processor.
const (
	driverOriginROM = 0x00000000
	driverMemtopROM = 0x00000bff

	customOriginROM = 0x00000c00
	customMemtopROM = 0x00006bff

	dataOriginROM = 0x00006c00
	dataMemtopROM = 0x00007bff

	freqOriginROM = 0x00007c00
	freqMemtopROM = 0x00008000

	driverOriginRAM = 0x40000000
	driverMemtopRAM = 0x40000bff

	dataOriginRAM = 0x40000c00
	dataMemtopRAM = 0x40001bff

	freqOriginRAM = 0x40001c00
	freqMemtopRAM = 0x40002000

	// stack should be within the range of the RAM copy of the frequency tables
	stackOriginRAM = 0x40001fdc
)

// MapAddress implements the arm7tdmi.SharedMemory interface.
func (mem *DPCplusStatic) MapAddress(addr uint32, write bool) (*[]byte, uint32) {
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

	s[0].Data = make([]byte, len(cart.static.driverRAM))
	s[1].Data = make([]byte, len(cart.static.dataRAM))
	s[2].Data = make([]byte, len(cart.static.freqRAM))

	copy(s[0].Data, cart.static.driverRAM)
	copy(s[1].Data, cart.static.dataRAM)
	copy(s[1].Data, cart.static.freqRAM)

	return s
}

// StaticWrite implements the bus.CartDebugBus interface.
func (cart *dpcPlus) PutStatic(segment string, idx uint16, data uint8) error {
	switch segment {
	case "Driver":
		if int(idx) >= len(cart.static.driverRAM) {
			return curated.Errorf("CDFJ", fmt.Errorf("index too high (%#04x) for %s area", idx, segment))
		}
		cart.static.driverRAM[idx] = data

	case "Data":
		if int(idx) >= len(cart.static.dataRAM) {
			return curated.Errorf("DPC+: static: %v", fmt.Errorf("index too high (%#04x) for %s area", idx, segment))
		}
		cart.static.dataRAM[idx] = data

	case "Freq":
		if int(idx) >= len(cart.static.freqRAM) {
			return curated.Errorf("DPC+: static: %v", fmt.Errorf("index too high (%#04x) for %s area", idx, segment))
		}
		cart.static.freqRAM[idx] = data

	default:
		return curated.Errorf("DPC+: static: %v", fmt.Errorf("unknown segment (%s)", segment))
	}

	return nil
}
