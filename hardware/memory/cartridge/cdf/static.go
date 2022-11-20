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
	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/logger"
)

// Static implements the mapper.CartStatic interface.
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

func (cart *cdf) newCDFstatic(instance *instance.Instance, cartData []byte, r version) *Static {
	stc := Static{version: r}

	// ARM driver
	stc.driverROM = cartData[:driverSize]

	// custom ARM program begins immediately after the ARM driver
	stc.customROM = cartData[stc.version.customOriginROM-stc.version.mmap.FlashOrigin:]

	// driver RAM is the same as driver ROM initially
	stc.driverRAM = make([]byte, driverSize)
	copy(stc.driverRAM, stc.driverROM)

	// there is nothing in cartData to copy into the other RAM areas
	stc.dataRAM = make([]byte, stc.version.dataMemtopRAM-stc.version.dataOriginRAM+1)
	stc.variablesRAM = make([]byte, stc.version.variablesMemtopRAM-stc.version.variablesOriginRAM+1)

	if instance.Prefs.RandomState.Get().(bool) {
		for i := range stc.dataRAM {
			stc.dataRAM[i] = uint8(instance.Random.NoRewind(0xff))
		}
		for i := range stc.variablesRAM {
			stc.variablesRAM[i] = uint8(instance.Random.NoRewind(0xff))
		}
	}

	return &stc
}

func (stc *Static) HotLoad(cartData []byte) {
	// ARM driver
	stc.driverROM = cartData[:driverSize]

	// custom ARM program
	stc.customROM = cartData[stc.version.customOriginROM-stc.version.mmap.FlashOrigin:]
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

	// data (RAM)
	if addr >= stc.version.dataOriginRAM && addr <= stc.version.dataMemtopRAM {
		return &stc.dataRAM, addr - stc.version.dataOriginRAM
	}

	// variables (RAM)
	if addr >= stc.version.variablesOriginRAM && addr <= stc.version.variablesMemtopRAM {
		return &stc.variablesRAM, addr - stc.version.variablesOriginRAM
	}

	// custom ARM code (ROM)
	if addr >= stc.version.customOriginROM && addr <= stc.version.customMemtopROM {
		if write {
			logger.Logf("CDF", "ARM trying to write to custom ROM address (%08x)", addr)
			return nil, addr
		}
		return &stc.customROM, addr - stc.version.customOriginROM
	}

	// driver ARM code (RAM)
	if addr >= stc.version.driverOriginRAM && addr <= stc.version.driverMemtopRAM {
		return &stc.driverRAM, addr - stc.version.driverOriginRAM
	}

	// driver ARM code (ROM)
	if addr >= stc.version.driverOriginROM && addr <= stc.version.driverMemtopROM {
		if write {
			logger.Logf("CDF", "ARM trying to write to driver ROM address (%08x)", addr)
			return nil, addr
		}
		return &stc.driverROM, addr - stc.version.driverOriginROM
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
			Name:   "Variables",
			Origin: stc.version.variablesOriginRAM,
			Memtop: stc.version.variablesMemtopRAM,
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
	case "Variables":
		return stc.variablesRAM, true
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

// GetStatic implements the mapper.CartStaticBus interface.
func (cart *cdf) GetStatic() mapper.CartStatic {
	return cart.state.static.Snapshot()
}

// StaticWrite implements the mapper.CartStaticBus interface.
func (cart *cdf) PutStatic(segment string, idx int, data uint8) bool {
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

	case "Variables":
		if idx >= len(cart.state.static.variablesRAM) {
			return false
		}
		cart.state.static.variablesRAM[idx] = data

	default:
		return false
	}

	return true
}
