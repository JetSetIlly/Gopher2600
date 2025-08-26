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

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

type segment struct {
	name   string
	data   []byte
	origin uint32
	memtop uint32
}

func (seg segment) snapshot() segment {
	n := seg
	n.data = make([]byte, len(seg.data))
	copy(n.data, seg.data)
	return n
}

// Static implements the mapper.CartStatic interface.
type Static struct {
	version version

	// slices of cartDataROM that should not be modified during execution
	driverROM segment
	customROM segment
	dataROM   segment
	freqROM   segment

	// slices of cartDataRAM that will be modified during execution
	driverRAM segment
	dataRAM   segment
	freqRAM   segment
}

func (cart *dpcPlus) newDPCplusStatic(version version, cartData []byte) (*Static, error) {
	stc := Static{
		version: version,
	}

	// the offset into the cart data where the data segment begins
	dataOffset := driverSize + (cart.bankSize * cart.NumBanks())

	// ARM driver
	stc.driverROM.name = "Driver ROM"
	stc.driverROM.data = cartData[:driverSize]
	stc.driverROM.origin = version.driverROMOrigin
	stc.driverROM.memtop = version.driverROMOrigin + uint32(len(stc.driverROM.data)) - 1
	if stc.driverROM.memtop > version.driverROMMemtop {
		return nil, fmt.Errorf("driver ROM is too large")
	}

	// custom ARM program immediately after the ARM driver and where we've
	// figured the data segment to start. note that some of this will be the
	// 6507 program but we can't really know for sure where that begins.
	stc.customROM.name = "Custom ROM"
	stc.customROM.data = cartData[driverSize:dataOffset]
	stc.customROM.origin = version.customROMOrigin
	stc.customROM.memtop = version.customROMOrigin + uint32(len(stc.customROM.data)) - 1
	if stc.customROM.memtop > version.customROMMemtop {
		return nil, fmt.Errorf("custom ROM is too large")
	}

	// gfx and frequency table at end of file
	// unlike CDF ROMs data and frequency tables are initialised from the ROM

	stc.dataROM.name = "Data ROM"
	stc.dataROM.data = cartData[dataOffset : dataOffset+dataSize]
	stc.dataROM.origin = version.dataROMOrigin
	stc.dataROM.memtop = version.dataROMOrigin + uint32(len(stc.dataROM.data)) - 1
	if stc.dataROM.memtop > version.dataROMMemtop {
		return nil, fmt.Errorf("data ROM is too large")
	}

	stc.freqROM.name = "Freq ROM"
	stc.freqROM.data = cartData[dataOffset+dataSize:]
	stc.freqROM.origin = version.freqROMOrigin
	stc.freqROM.memtop = version.freqROMOrigin + uint32(len(stc.freqROM.data)) - 1
	if stc.freqROM.memtop > version.freqROMMemtop {
		return nil, fmt.Errorf("freq ROM is too large")
	}

	// RAM areas. because these areas are the ones that we are likely to show
	// most often in UI, the RAM suffix has been omitted

	stc.driverRAM.name = "Driver"
	stc.driverRAM.data = make([]byte, len(stc.driverROM.data))
	stc.driverRAM.origin = version.driverRAMOrigin
	stc.driverRAM.memtop = version.driverRAMOrigin + uint32(len(stc.driverRAM.data)) - 1
	if stc.driverRAM.memtop > version.driverRAMMemtop {
		return nil, fmt.Errorf("driver RAM is too large")
	}
	copy(stc.driverRAM.data, stc.driverROM.data)

	stc.dataRAM.name = "Data"
	stc.dataRAM.data = make([]byte, len(stc.dataROM.data))
	stc.dataRAM.origin = version.dataRAMOrigin
	stc.dataRAM.memtop = version.dataRAMOrigin + uint32(len(stc.dataRAM.data)) - 1
	if stc.dataRAM.memtop > version.dataRAMMemtop {
		return nil, fmt.Errorf("data RAM is too large")
	}
	copy(stc.dataRAM.data, stc.dataROM.data)

	stc.freqRAM.name = "Freq"
	stc.freqRAM.data = make([]byte, len(stc.freqROM.data))
	stc.freqRAM.origin = version.freqRAMOrigin
	stc.freqRAM.memtop = version.freqRAMOrigin + uint32(len(stc.freqRAM.data)) - 1
	if stc.freqRAM.memtop > version.freqRAMMemtop {
		return nil, fmt.Errorf("freq RAM is too large")
	}
	copy(stc.freqRAM.data, stc.freqROM.data)

	return &stc, nil
}

// ResetVectors implements the arm.SharedMemory interface.
func (stc *Static) ResetVectors() (uint32, uint32, uint32) {
	return stc.version.stackOrigin, stc.customROM.origin, stc.customROM.origin + 8
}

// IsExecutable implements the arm.SharedMemory interface.
func (mem *Static) IsExecutable(addr uint32) bool {
	return true
}

func (stc *Static) Snapshot() *Static {
	n := *stc
	n.driverRAM = stc.driverRAM.snapshot()
	n.dataRAM = stc.dataRAM.snapshot()
	n.freqRAM = stc.freqRAM.snapshot()
	return &n
}

// MapAddress implements the arm.SharedMemory interface.
func (stc *Static) MapAddress(addr uint32, write bool, executing bool) (*[]byte, uint32) {
	// tests arranged in order of most likely to be used. determined by running
	// ZaxxonHDDemo through the go profiler

	// data (RAM)
	if addr >= stc.dataRAM.origin && addr <= stc.dataRAM.memtop {
		return &stc.dataRAM.data, stc.dataRAM.origin
	}

	// custom ARM code (ROM)
	if addr >= stc.customROM.origin && addr <= stc.customROM.memtop {
		if write {
			return nil, 0
		}
		return &stc.customROM.data, stc.customROM.origin
	}

	// driver ARM code (RAM)
	if addr >= stc.driverRAM.origin && addr <= stc.driverRAM.memtop {
		return &stc.driverRAM.data, stc.driverRAM.origin
	}

	// frequency table (RAM)
	if addr >= stc.freqRAM.origin && addr <= stc.freqRAM.memtop {
		return &stc.freqRAM.data, stc.freqRAM.origin
	}

	// driver ARM code (ROM)
	if addr >= stc.driverROM.origin && addr <= stc.driverROM.memtop {
		if write {
			return nil, 0
		}
		return &stc.driverROM.data, stc.driverROM.origin
	}

	// data (ROM)
	if addr >= stc.dataROM.origin && addr <= stc.dataROM.memtop {
		if write {
			return nil, 0
		}
		return &stc.dataROM.data, stc.dataROM.origin
	}

	// frequency table (ROM)
	if addr >= stc.freqROM.origin && addr <= stc.freqROM.memtop {
		if write {
			return nil, 0
		}
		return &stc.freqROM.data, stc.freqROM.origin
	}

	return nil, 0
}

// Segments implements the mapper.CartStatic interface
func (stc *Static) Segments() []mapper.CartStaticSegment {
	return []mapper.CartStaticSegment{
		{
			Name:   stc.driverRAM.name,
			Origin: stc.driverRAM.origin,
			Memtop: stc.driverRAM.memtop,
		},
		{
			Name:   stc.dataRAM.name,
			Origin: stc.dataRAM.origin,
			Memtop: stc.dataRAM.memtop,
		},
		{
			Name:   stc.freqRAM.name,
			Origin: stc.freqRAM.origin,
			Memtop: stc.freqRAM.memtop,
		},
	}
}

// Reference implements the mapper.CartStatic interface
func (stc *Static) Reference(segment string) ([]uint8, bool) {
	switch segment {
	case stc.driverRAM.name:
		return stc.driverRAM.data, true
	case stc.dataRAM.name:
		return stc.dataRAM.data, true
	case stc.freqRAM.name:
		return stc.freqRAM.data, true
	}
	return []uint8{}, false
}

// Read8bit implements the mapper.CartStatic interface
func (stc *Static) Read8bit(addr uint32) (uint8, bool) {
	mem, origin := stc.MapAddress(addr, false, false)
	addr -= origin
	if mem == nil || addr >= uint32(len(*mem)) {
		return 0, false
	}
	return (*mem)[addr], true
}

// Read16bit implements the mapper.CartStatic interface
func (stc *Static) Read16bit(addr uint32) (uint16, bool) {
	mem, origin := stc.MapAddress(addr, false, false)
	addr -= origin
	if mem == nil || addr >= uint32(len(*mem)-1) {
		return 0, false
	}
	return uint16((*mem)[addr]) |
		uint16((*mem)[addr+1])<<8, true
}

// Read32bit implements the mapper.CartStatic interface
func (stc *Static) Read32bit(addr uint32) (uint32, bool) {
	mem, origin := stc.MapAddress(addr, false, false)
	addr -= origin
	if mem == nil || addr >= uint32(len(*mem)-3) {
		return 0, false
	}
	return uint32((*mem)[addr]) |
		uint32((*mem)[addr+1])<<8 |
		uint32((*mem)[addr+2])<<16 |
		uint32((*mem)[addr+3])<<24, true
}

// Read8bit implements the mapper.CartStatic interface
func (stc *Static) Write8bit(addr uint32, data uint8) bool {
	mem, origin := stc.MapAddress(addr, false, false)
	addr -= origin
	if mem == nil || addr >= uint32(len(*mem)) {
		return false
	}
	(*mem)[addr] = data
	return true
}

// GetStatic implements the mapper.CartStaticBus interface
func (cart *dpcPlus) GetStatic() mapper.CartStatic {
	return cart.state.static.Snapshot()
}

// ReferenceStatic implements the mapper.CartStaticBus interface.
func (cart *dpcPlus) ReferenceStatic() mapper.CartStatic {
	return cart.state.static
}

// StaticWrite implements the mapper.CartStaticBus interface
func (cart *dpcPlus) PutStatic(segment string, idx int, data uint8) bool {
	switch segment {
	case cart.state.static.driverRAM.name:
		if idx >= len(cart.state.static.driverRAM.data) {
			return false
		}
		cart.state.static.driverRAM.data[idx] = data

	case cart.state.static.dataRAM.name:
		if idx >= len(cart.state.static.dataRAM.data) {
			return false
		}
		cart.state.static.dataRAM.data[idx] = data

	case cart.state.static.freqRAM.name:
		if idx >= len(cart.state.static.freqRAM.data) {
			return false
		}
		cart.state.static.freqRAM.data[idx] = data

	default:
		return false
	}

	return true
}
