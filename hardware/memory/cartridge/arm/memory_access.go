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

package arm

import (
	"errors"
	"fmt"

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/faults"
	"github.com/jetsetilly/gopher2600/logger"
)

func (arm *ARM) illegalAccess(event string, addr uint32) {
	if arm.state.stackHasCollided {
		return
	}

	arm.state.yield.Type = coprocessor.YieldMemoryAccessError
	arm.state.yield.Error = fmt.Errorf("%s: unrecognised address %08x (PC: %08x)", event, addr, arm.state.instructionPC)

	if arm.dev == nil {
		return
	}

	detail := arm.dev.MemoryFault(event, faults.IllegalAddress, arm.state.instructionPC, addr)
	if detail != "" {
		arm.state.yield.Detail = append(arm.state.yield.Detail, errors.New(detail))
	}
}

// nullAccess is a special condition of illegalAccess()
func (arm *ARM) nullAccess(event string, addr uint32) {
	arm.state.yield.Type = coprocessor.YieldMemoryAccessError
	arm.state.yield.Error = fmt.Errorf("%s: probable null pointer dereference of %08x (PC: %08x)", event, addr, arm.state.instructionPC)

	if arm.dev == nil {
		return
	}

	detail := arm.dev.MemoryFault(event, faults.NullDereference, arm.state.instructionPC, addr)
	if detail != "" {
		arm.state.yield.Detail = append(arm.state.yield.Detail, errors.New(detail))
	}
}

func (arm *ARM) read8bit(addr uint32) uint8 {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Read 8bit", addr)
	}

	mem, origin := arm.mem.MapAddress(addr, false)
	if mem == nil {
		if arm.mmap.HasMAM {
			if v, ok, comment := arm.state.mam.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint8(v)
			}
		}
		if arm.mmap.HasRNG {
			if v, ok, comment := arm.state.rng.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint8(v)
			}
		}
		if arm.mmap.HasTIMER {
			if v, ok, comment := arm.state.timer.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint8(v)
			}
		}
		if arm.mmap.HasTIM2 {
			if v, ok, comment := arm.state.timer2.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint8(v)
			}
		}

		arm.illegalAccess("Read 8bit", addr)
		return uint8(arm.mmap.IllegalAccessValue)
	}

	// adjust address so that it can be used as an index
	idx := addr - origin
	return (*mem)[idx]

}

func (arm *ARM) write8bit(addr uint32, val uint8) {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Write 8bit", addr)
	}

	mem, origin := arm.mem.MapAddress(addr, true)
	if mem == nil {
		if arm.mmap.HasMAM {
			if ok, comment := arm.state.mam.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasRNG {
			if ok, comment := arm.state.rng.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasTIMER {
			if ok, comment := arm.state.timer.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasTIM2 {
			if ok, comment := arm.state.timer2.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}

		arm.illegalAccess("Write 8bit", addr)
		return
	}

	// adjust address so that it can be used as an index
	idx := addr - origin
	(*mem)[idx] = val
}

// requiresAlignment should be true only for certain instructions. alignment
// behaviour given in "A63.2.1 Alignment behaviour" of "ARMv7-M"
func (arm *ARM) read16bit(addr uint32, requiresAlignment bool) uint16 {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Read 16bit", addr)
	}

	// check 16 bit alignment
	if requiresAlignment && addr&0x01 != 0x00 {
		logger.Logf("ARM7", "misaligned 16 bit read (%08x) (PC: %08x)", addr, arm.state.instructionPC)
	}

	mem, origin := arm.mem.MapAddress(addr, false)
	if mem == nil {
		if arm.mmap.HasMAM {
			if v, ok, comment := arm.state.mam.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint16(v)
			}
		}
		if arm.mmap.HasRNG {
			if v, ok, comment := arm.state.rng.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint16(v)
			}
		}
		if arm.mmap.HasTIMER {
			if v, ok, comment := arm.state.timer.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint16(v)
			}
		}
		if arm.mmap.HasTIM2 {
			if v, ok, comment := arm.state.timer2.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint16(v)
			}
		}

		arm.illegalAccess("Read 16bit", addr)
		return uint16(arm.mmap.IllegalAccessValue)
	}

	// adjust address so that it can be used as an index
	idx := addr - origin

	// ensure we're not accessing past the end of memory
	if len(*mem) < 2 || idx >= uint32(len(*mem)-1) {
		arm.illegalAccess("Read 16bit", addr)
		return uint16(arm.mmap.IllegalAccessValue)
	}

	return arm.byteOrder.Uint16((*mem)[idx:])
}

// requiresAlignment should be true only for certain instructions. alignment
// behaviour given in "A63.2.1 Alignment behaviour" of "ARMv7-M"
func (arm *ARM) write16bit(addr uint32, val uint16, requiresAlignment bool) {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Write 16bit", addr)
	}

	// check 16 bit alignment
	if requiresAlignment && addr&0x01 != 0x00 {
		logger.Logf("ARM7", "misaligned 16 bit write (%08x) (PC: %08x)", addr, arm.state.instructionPC)
	}

	mem, origin := arm.mem.MapAddress(addr, true)
	if mem == nil {
		if arm.mmap.HasMAM {
			if ok, comment := arm.state.mam.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasRNG {
			if ok, comment := arm.state.rng.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasTIMER {
			if ok, comment := arm.state.timer.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasTIM2 {
			if ok, comment := arm.state.timer2.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}

		arm.illegalAccess("Write 16bit", addr)
		return
	}

	// adjust address so that it can be used as an index
	idx := addr - origin

	// ensure we're not accessing past the end of memory
	if len(*mem) < 2 || idx >= uint32(len(*mem)-1) {
		arm.illegalAccess("Write 16bit", addr)
		return
	}

	arm.byteOrder.PutUint16((*mem)[idx:], val)
}

// requiresAlignment should be true only for certain instructions. alignment
// behaviour given in "A63.2.1 Alignment behaviour" of "ARMv7-M"
func (arm *ARM) read32bit(addr uint32, requiresAlignment bool) uint32 {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Read 32bit", addr)
	}

	// check 32 bit alignment
	if requiresAlignment && addr&0x03 != 0x00 {
		logger.Logf("ARM7", "misaligned 32 bit read (%08x) (PC: %08x)", addr, arm.state.instructionPC)
	}

	mem, origin := arm.mem.MapAddress(addr, false)
	if mem == nil {
		if arm.mmap.HasMAM {
			if v, ok, comment := arm.state.mam.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint32(v)
			}
		}
		if arm.mmap.HasRNG {
			if v, ok, comment := arm.state.rng.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint32(v)
			}
		}
		if arm.mmap.HasTIMER {
			if v, ok, comment := arm.state.timer.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint32(v)
			}
		}
		if arm.mmap.HasTIM2 {
			if v, ok, comment := arm.state.timer2.Read(addr); ok {
				arm.disasmExecutionNotes = comment
				return uint32(v)
			}
		}

		arm.illegalAccess("Read 32bit", addr)
		return arm.mmap.IllegalAccessValue
	}

	// adjust address so that it can be used as an index
	idx := addr - origin

	// ensure we're not accessing past the end of memory
	if len(*mem) < 4 || idx >= uint32(len(*mem)-3) {
		arm.illegalAccess("Read 32bit", addr)
		return arm.mmap.IllegalAccessValue
	}

	return arm.byteOrder.Uint32((*mem)[idx:])
}

// requiresAlignment should be true only for certain instructions. alignment
// behaviour given in "A63.2.1 Alignment behaviour" of "ARMv7-M"
func (arm *ARM) write32bit(addr uint32, val uint32, requiresAlignment bool) {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Write 32bit", addr)
	}

	// check 32 bit alignment
	if requiresAlignment && addr&0x03 != 0x00 {
		logger.Logf("ARM7", "misaligned 32 bit write (%08x) (PC: %08x)", addr, arm.state.instructionPC)
	}

	mem, origin := arm.mem.MapAddress(addr, true)
	if mem == nil {
		if arm.mmap.HasMAM {
			if ok, comment := arm.state.mam.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasRNG {
			if ok, comment := arm.state.rng.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasTIMER {
			if ok, comment := arm.state.timer.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}
		if arm.mmap.HasTIM2 {
			if ok, comment := arm.state.timer2.Write(addr, uint32(val)); ok {
				arm.disasmExecutionNotes = comment
				return
			}
		}

		arm.illegalAccess("Write 32bit", addr)
		return
	}

	// adjust address so that it can be used as an index
	idx := addr - origin

	// ensure we're not accessing past the end of memory
	if len(*mem) < 4 || idx >= uint32(len(*mem)-3) {
		arm.illegalAccess("Write 32bit", addr)
		return
	}

	arm.byteOrder.PutUint32((*mem)[idx:], val)
}

// Peek implements the coprocessor.CoProc interface
func (arm *ARM) Peek(addr uint32) (uint32, bool) {
	mem, origin := arm.mem.MapAddress(addr, false)
	addr -= origin
	if mem == nil || addr >= uint32(len(*mem)-3) {
		return 0, false
	}
	return arm.byteOrder.Uint32((*mem)[addr:]), true
}
