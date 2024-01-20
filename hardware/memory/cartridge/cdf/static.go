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

	"github.com/jetsetilly/gopher2600/environment"
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

	// slices of cartData that should not be modified during execution
	driverROM segment
	customROM segment

	// slices of cartData that will be modified during execution
	driverRAM segment
	dataRAM   segment

	// variablesRAM might be absent in the case of CDFJ+
	variablesRAM segment
}

func (cart *cdf) newCDFstatic(env *environment.Environment, version version, cartData []byte) (*Static, error) {
	stc := Static{
		version: version,
	}

	// ARM driver
	stc.driverROM.name = "Driver ROM"
	stc.driverROM.data = cartData[:driverSize]
	stc.driverROM.origin = stc.version.driverROMOrigin
	stc.driverROM.memtop = stc.version.driverROMOrigin + uint32(len(stc.driverROM.data)) - 1
	if stc.driverROM.memtop > stc.version.driverROMMemtop {
		return nil, fmt.Errorf("driver ROM is too large")
	}

	// custom ARM program begins immediately after the ARM driver
	stc.customROM.name = "Custom ROM"
	stc.customROM.data = cartData[stc.version.customROMOrigin-stc.version.mmap.FlashOrigin:]
	stc.customROM.origin = stc.version.customROMOrigin
	stc.customROM.memtop = stc.version.customROMOrigin + uint32(len(stc.customROM.data)) - 1
	if stc.customROM.memtop > stc.version.customROMMemtop {
		return nil, fmt.Errorf("custom ROM is too large")
	}

	// RAM areas. because these areas are the ones that we are likely to show
	// most often in UI, the RAM suffix has been omitted

	stc.driverRAM.name = "Driver"
	stc.driverRAM.data = make([]byte, len(stc.driverROM.data))
	stc.driverRAM.origin = stc.version.driverRAMOrigin
	stc.driverRAM.memtop = stc.version.driverRAMOrigin + uint32(len(stc.driverRAM.data)) - 1
	if stc.driverRAM.memtop > stc.version.driverRAMMemtop {
		return nil, fmt.Errorf("driver RAM is too large")
	}
	copy(stc.driverRAM.data, stc.driverROM.data)

	stc.dataRAM.name = "Data"
	stc.dataRAM.data = make([]byte, stc.version.dataRAMMemtop-stc.version.dataRAMOrigin+1)
	stc.dataRAM.origin = stc.version.dataRAMOrigin
	stc.dataRAM.memtop = stc.version.dataRAMMemtop

	// variables ram is not used in CDFJ+
	if stc.version.variablesRAMMemtop > stc.version.variablesRAMOrigin {
		stc.variablesRAM.name = "Variables"
		stc.variablesRAM.data = make([]byte, stc.version.variablesRAMMemtop-stc.version.variablesRAMOrigin+1)
		stc.variablesRAM.origin = stc.version.variablesRAMOrigin
		stc.variablesRAM.memtop = stc.version.variablesRAMMemtop
	}

	// randomise initial state if preference is set
	if env.Prefs.RandomState.Get().(bool) {
		for i := range stc.dataRAM.data {
			stc.dataRAM.data[i] = uint8(env.Random.NoRewind(0xff))
		}

		if stc.hasVariableSegment() {
			for i := range stc.variablesRAM.data {
				stc.variablesRAM.data[i] = uint8(env.Random.NoRewind(0xff))
			}
		}
	}

	return &stc, nil
}

func (stc *Static) hasVariableSegment() bool {
	return stc.variablesRAM.name != ""
}

// ResetVectors implements the arm7tdmi.SharedMemory interface.
func (stc *Static) ResetVectors() (uint32, uint32, uint32) {
	return stc.version.entrySP, stc.version.entryLR, stc.version.entryPC
}

// IsExecutable implements the arm.SharedMemory interface.
func (mem *Static) IsExecutable(addr uint32) bool {
	return true
}

func (stc *Static) Snapshot() *Static {
	n := *stc
	n.driverRAM = stc.driverRAM.snapshot()
	n.dataRAM = stc.dataRAM.snapshot()
	n.variablesRAM = stc.variablesRAM.snapshot()
	return &n
}

// MapAddress implements the arm7tdmi.SharedMemory interface.
func (stc *Static) MapAddress(addr uint32, write bool) (*[]byte, uint32) {
	// tests arranged in order of most likely to be used

	// data (RAM)
	if addr >= stc.dataRAM.origin && addr <= stc.dataRAM.memtop {
		return &stc.dataRAM.data, stc.dataRAM.origin
	}

	// variables (RAM)
	if stc.hasVariableSegment() {
		if addr >= stc.variablesRAM.origin && addr <= stc.variablesRAM.memtop {
			return &stc.variablesRAM.data, stc.variablesRAM.origin
		}
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

	// driver ARM code (ROM)
	if addr >= stc.driverROM.origin && addr <= stc.driverROM.memtop {
		if write {
			return nil, 0
		}
		return &stc.driverROM.data, stc.driverROM.origin
	}

	return nil, 0
}

// Segments implements the mapper.CartStatic interface
func (stc *Static) Segments() []mapper.CartStaticSegment {
	segments := []mapper.CartStaticSegment{
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
	}
	if stc.hasVariableSegment() {
		segments = append(segments, mapper.CartStaticSegment{
			Name:   stc.variablesRAM.name,
			Origin: stc.variablesRAM.origin,
			Memtop: stc.variablesRAM.memtop,
		})
	}
	return segments
}

// Reference implements the mapper.CartStatic interface
func (stc *Static) Reference(segment string) ([]uint8, bool) {
	switch segment {
	case stc.driverRAM.name:
		return stc.driverRAM.data, true
	case stc.dataRAM.name:
		return stc.dataRAM.data, true
	case stc.variablesRAM.name:
		return stc.variablesRAM.data, stc.hasVariableSegment()
	}
	return []uint8{}, false
}

// Read8bit implements the mapper.CartStatic interface
func (stc *Static) Read8bit(addr uint32) (uint8, bool) {
	mem, origin := stc.MapAddress(addr, false)
	addr -= origin
	if mem == nil || addr >= uint32(len(*mem)) {
		return 0, false
	}
	return (*mem)[addr], true
}

// Read16bit implements the mapper.CartStatic interface
func (stc *Static) Read16bit(addr uint32) (uint16, bool) {
	mem, origin := stc.MapAddress(addr, false)
	addr -= origin
	if mem == nil || addr >= uint32(len(*mem)-1) {
		return 0, false
	}
	return uint16((*mem)[addr]) |
		uint16((*mem)[addr+1])<<8, true
}

// Read32bit implements the mapper.CartStatic interface
func (stc *Static) Read32bit(addr uint32) (uint32, bool) {
	mem, origin := stc.MapAddress(addr, false)
	addr -= origin
	if mem == nil || addr >= uint32(len(*mem)-3) {
		return 0, false
	}
	return uint32((*mem)[addr]) |
		uint32((*mem)[addr+1])<<8 |
		uint32((*mem)[addr+2])<<16 |
		uint32((*mem)[addr+3])<<24, true
}

// GetStatic implements the mapper.CartStaticBus interface.
func (cart *cdf) GetStatic() mapper.CartStatic {
	return cart.state.static.Snapshot()
}

// StaticWrite implements the mapper.CartStaticBus interface.
func (cart *cdf) PutStatic(segment string, idx int, data uint8) bool {
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

	case cart.state.static.variablesRAM.name:
		if cart.state.static.hasVariableSegment() {
			if idx >= len(cart.state.static.variablesRAM.data) {
				return false
			}
			cart.state.static.variablesRAM.data[idx] = data
		}

	default:
		return false
	}

	return true
}
