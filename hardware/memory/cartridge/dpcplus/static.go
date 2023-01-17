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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

// Static implements the mapper.CartStatic interface.
type Static struct {
	version version

	// slices of cartDataROM that should not be modified during execution
	driverROM []byte
	customROM []byte
	dataROM   []byte
	freqROM   []byte

	// slices of cartDataRAM that will be modified during execution
	driverRAM []byte
	dataRAM   []byte
	freqRAM   []byte
}

func (cart *dpcPlus) newDPCplusStatic(version version, cartData []byte) *Static {
	stc := Static{
		version: version,
	}

	// the offset into the cart data where the data segment begins
	dataOffset := driverSize + (cart.bankSize * cart.NumBanks())

	// ARM driver
	stc.driverROM = cartData[:driverSize]

	// custom ARM program immediately after the ARM driver and where we've
	// figured the data segment to start. note that some of this will be the
	// 6507 program but we can't really know for sure where that begins.
	stc.customROM = cartData[driverSize:dataOffset]

	// gfx and frequency table at end of file
	// unlike CDF ROMs data and frequency tables are initialised from the ROM
	stc.dataROM = cartData[dataOffset : dataOffset+dataSize]
	stc.freqROM = cartData[dataOffset+dataSize:]

	// RAM areas
	stc.driverRAM = make([]byte, len(stc.driverROM))
	stc.dataRAM = make([]byte, len(stc.dataROM))
	stc.freqRAM = make([]byte, len(stc.freqROM))
	copy(stc.driverRAM, stc.driverROM)
	copy(stc.dataRAM, stc.dataROM)
	copy(stc.freqRAM, stc.freqROM)

	return &stc
}

// ResetVectors implements the arm7tdmi.SharedMemory interface.
func (stc *Static) ResetVectors() (uint32, uint32, uint32) {
	return stc.version.stackOriginRAM, stc.version.customOriginROM, stc.version.customOriginROM + 8
}

// IsExecutable implements the arm.SharedMemory interface.
func (mem *Static) IsExecutable(addr uint32) bool {
	return true
}

func (stc *Static) Snapshot() *Static {
	n := *stc
	n.driverRAM = make([]byte, len(stc.driverRAM))
	n.dataRAM = make([]byte, len(stc.dataRAM))
	n.freqRAM = make([]byte, len(stc.freqRAM))
	copy(n.driverRAM, stc.driverRAM)
	copy(n.dataRAM, stc.dataRAM)
	copy(n.freqRAM, stc.freqRAM)
	return &n
}

// MapAddress implements the arm7tdmi.SharedMemory interface.
func (stc *Static) MapAddress(addr uint32, write bool) (*[]byte, uint32) {
	// tests arranged in order of most likely to be used. determined by running
	// ZaxxonHDDemo through the go profiler

	// data (RAM)
	if addr >= stc.version.dataOriginRAM && addr <= stc.version.dataMemtopRAM {
		return &stc.dataRAM, addr - stc.version.dataOriginRAM
	}

	// custom ARM code (ROM)
	if addr >= stc.version.customOriginROM && addr <= stc.version.customMemtopROM {
		if write {
			return nil, addr
		}
		return &stc.customROM, addr - stc.version.customOriginROM
	}

	// driver ARM code (RAM)
	if addr >= stc.version.driverOriginRAM && addr <= stc.version.driverMemtopRAM {
		return &stc.driverRAM, addr - stc.version.driverOriginRAM
	}

	// frequency table (RAM)
	if addr >= stc.version.freqOriginRAM && addr <= stc.version.freqMemtopRAM {
		return &stc.freqRAM, addr - stc.version.freqOriginRAM
	}

	// driver ARM code (ROM)
	if addr >= stc.version.driverOriginROM && addr <= stc.version.driverMemtopROM {
		if write {
			return nil, addr
		}
		return &stc.driverROM, addr - stc.version.driverOriginROM
	}

	// data (ROM)
	if addr >= stc.version.dataOriginROM && addr <= stc.version.dataMemtopROM {
		if write {
			return nil, addr
		}
		return &stc.dataROM, addr - stc.version.dataOriginROM
	}

	// frequency table (ROM)
	if addr >= stc.version.freqOriginROM && addr <= stc.version.freqMemtopROM {
		if write {
			return nil, addr
		}
		return &stc.freqROM, addr - stc.version.freqOriginROM
	}

	return nil, addr
}

// Segments implements the mapper.CartStatic interface
func (stc *Static) Segments() []mapper.CartStaticSegment {
	return []mapper.CartStaticSegment{
		mapper.CartStaticSegment{
			Name:   "Driver",
			Origin: stc.version.driverOriginRAM,
			Memtop: stc.version.driverMemtopRAM,
		},
		mapper.CartStaticSegment{
			Name:   "Data",
			Origin: stc.version.dataOriginRAM,
			Memtop: stc.version.dataMemtopRAM,
		},
		mapper.CartStaticSegment{
			Name:   "Frequencies",
			Origin: stc.version.freqOriginRAM,
			Memtop: stc.version.freqMemtopRAM,
		},
	}
}

// Reference implements the mapper.CartStatic interface
func (stc *Static) Reference(segment string) ([]uint8, bool) {
	switch segment {
	case "Driver":
		return stc.driverRAM, true
	case "Data":
		return stc.dataRAM, true
	case "Frequencies":
		return stc.freqRAM, true
	}
	return []uint8{}, false
}

// Read8bit implements the mapper.CartStatic interface
func (stc *Static) Read8bit(addr uint32) (uint8, bool) {
	mem, addr := stc.MapAddress(addr, false)
	if mem == nil || addr >= uint32(len(*mem)) {
		return 0, false
	}
	return (*mem)[addr], true
}

// Read16bit implements the mapper.CartStatic interface
func (stc *Static) Read16bit(addr uint32) (uint16, bool) {
	mem, addr := stc.MapAddress(addr, false)
	if mem == nil || addr >= uint32(len(*mem)-1) {
		return 0, false
	}
	return uint16((*mem)[addr]) |
		uint16((*mem)[addr+1])<<8, true
}

// Read32bit implements the mapper.CartStatic interface
func (stc *Static) Read32bit(addr uint32) (uint32, bool) {
	mem, addr := stc.MapAddress(addr, false)
	if mem == nil || addr >= uint32(len(*mem)-3) {
		return 0, false
	}
	return uint32((*mem)[addr]) |
		uint32((*mem)[addr+1])<<8 |
		uint32((*mem)[addr+2])<<16 |
		uint32((*mem)[addr+3])<<24, true
}

// GetStatic implements the mapper.CartStaticBus interface
func (cart *dpcPlus) GetStatic() mapper.CartStatic {
	return cart.state.static.Snapshot()
}

// StaticWrite implements the mapper.CartStaticBus interface
func (cart *dpcPlus) PutStatic(segment string, idx int, data uint8) bool {
	switch segment {
	case "Driver":
		if idx >= len(cart.state.static.driverRAM) {
			return false
		}
		cart.state.static.driverRAM[idx] = data

	case "Data":
		if idx >= len(cart.state.static.dataRAM) {
			return false
		}
		cart.state.static.dataRAM[idx] = data

	case "Freq":
		if idx >= len(cart.state.static.freqRAM) {
			return false
		}
		cart.state.static.freqRAM[idx] = data

	default:
		return false
	}

	return true
}
