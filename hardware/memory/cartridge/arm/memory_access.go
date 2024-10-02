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
	"fmt"

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/faults"
)

func (arm *ARM) memoryFault(event string, fault faults.Category, addr uint32) {
	arm.state.yield.Type = coprocessor.YieldMemoryAccessError
	arm.state.yield.Error = fmt.Errorf("%s: %s: %08x (PC: %08x)", fault, event, addr, arm.state.instructionPC)

	if arm.dev == nil {
		return
	}

	arm.dev.MemoryFault(event, fault, arm.state.instructionPC, addr)
}

func (arm *ARM) illegalAccess(event string, addr uint32) {
	if arm.state.stackHasCollided {
		return
	}
	arm.memoryFault(event, faults.IllegalAddress, addr)
}

// nullAccess is a special condition of illegalAccess()
func (arm *ARM) nullAccess(event string, addr uint32) {
	arm.memoryFault(event, faults.NullDereference, addr)
}

// misalignedAccess is a special condition of illegalAccess()
func (arm *ARM) misalignedAccess(event string, addr uint32) {
	if arm.misalignedAccessIsFault {
		arm.memoryFault(event, faults.MisalignedAccess, addr)
	}
}

func (arm *ARM) read8bit(addr uint32) uint8 {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Read 8bit", addr)
	}

	mem, origin := arm.mem.MapAddress(addr, false, false)
	if mem == nil {
		if arm.mmap.HasMAM {
			if v, ok := arm.state.mam.Read(addr); ok {
				return uint8(v)
			}
		}
		if arm.mmap.HasRNG {
			if v, ok := arm.state.rng.Read(addr); ok {
				return uint8(v)
			}
		}
		if arm.mmap.HasT1 {
			if v, ok := arm.state.timer.Read(addr); ok {
				return uint8(v)
			}
		}
		if arm.mmap.HasTIM2 {
			if v, ok := arm.state.timer2.Read(addr); ok {
				return uint8(v)
			}
		}
		if addr == arm.mmap.APBDIV {
			return uint8(0)
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

	mem, origin := arm.mem.MapAddress(addr, true, false)
	if mem == nil {
		if arm.mmap.HasMAM {
			if arm.state.mam.Write(addr, uint32(val)) {
				return
			}
		}
		if arm.mmap.HasRNG {
			if arm.state.rng.Write(addr, uint32(val)) {
				return
			}
		}
		if arm.mmap.HasT1 {
			if arm.state.timer.Write(addr, uint32(val)) {
				return
			}
		}
		if arm.mmap.HasTIM2 {
			if arm.state.timer2.Write(addr, uint32(val)) {
				return
			}
		}
		if addr == arm.mmap.APBDIV {
			return
		}

		arm.illegalAccess("Write 8bit", addr)
		return
	}

	// adjust address so that it can be used as an index
	idx := addr - origin
	(*mem)[idx] = val
}

// for 16bit and 32bit access functions, there is a parameter called
// requiresAlignment. this indicates that the instruction issuing the access
// requires the access to be aligned.
//
// if the emulated architecture does not allow misaligned addresses then an
// appropriate alignment check is always made
//
// for the ARMv7-M architecture, alignment behaviour is given in "A63.2.1
// Alignment behaviour" of the specification

func (arm *ARM) read16bit(addr uint32, requiresAlignment bool) uint16 {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Read 16bit", addr)
	}

	// check 16 bit alignment
	if (requiresAlignment || !arm.mmap.MisalignedAccesses) && !IsAlignedTo16bits(addr) {
		arm.misalignedAccess("Read 16bit", addr)
		if !arm.mmap.MisalignedAccesses {
			addr = AlignTo16bits(addr)
		}
	}

	mem, origin := arm.mem.MapAddress(addr, false, false)
	if mem == nil {
		if arm.mmap.HasMAM {
			if v, ok := arm.state.mam.Read(addr); ok {
				return uint16(v)
			}
		}
		if arm.mmap.HasRNG {
			if v, ok := arm.state.rng.Read(addr); ok {
				return uint16(v)
			}
		}
		if arm.mmap.HasT1 {
			if v, ok := arm.state.timer.Read(addr); ok {
				return uint16(v)
			}
		}
		if arm.mmap.HasTIM2 {
			if v, ok := arm.state.timer2.Read(addr); ok {
				return uint16(v)
			}
		}
		if addr == arm.mmap.APBDIV {
			return uint16(0)
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

func (arm *ARM) write16bit(addr uint32, val uint16, requiresAlignment bool) {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Write 16bit", addr)
	}

	// check 16 bit alignment
	if (requiresAlignment || !arm.mmap.MisalignedAccesses) && !IsAlignedTo16bits(addr) {
		arm.misalignedAccess("Read 16bit", addr)
		if !arm.mmap.MisalignedAccesses {
			addr = AlignTo16bits(addr)
		}
	}

	mem, origin := arm.mem.MapAddress(addr, true, false)
	if mem == nil {
		if arm.mmap.HasMAM {
			if arm.state.mam.Write(addr, uint32(val)) {
				return
			}
		}
		if arm.mmap.HasRNG {
			if arm.state.rng.Write(addr, uint32(val)) {
				return
			}
		}
		if arm.mmap.HasT1 {
			if arm.state.timer.Write(addr, uint32(val)) {
				return
			}
		}
		if arm.mmap.HasTIM2 {
			if arm.state.timer2.Write(addr, uint32(val)) {
				return
			}
		}
		if addr == arm.mmap.APBDIV {
			return
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

func (arm *ARM) read32bit(addr uint32, requiresAlignment bool) uint32 {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Read 32bit", addr)
	}

	// check 32 bit alignment
	if (requiresAlignment || !arm.mmap.MisalignedAccesses) && !IsAlignedTo32bits(addr) {
		arm.misalignedAccess("Read 32bit", addr)
		if !arm.mmap.MisalignedAccesses {
			addr = AlignTo32bits(addr)
		}
	}

	mem, origin := arm.mem.MapAddress(addr, false, false)
	if mem == nil {
		if arm.mmap.HasMAM {
			if v, ok := arm.state.mam.Read(addr); ok {
				return uint32(v)
			}
		}
		if arm.mmap.HasRNG {
			if v, ok := arm.state.rng.Read(addr); ok {
				return uint32(v)
			}
		}
		if arm.mmap.HasT1 {
			if v, ok := arm.state.timer.Read(addr); ok {
				return uint32(v)
			}
		}
		if arm.mmap.HasTIM2 {
			if v, ok := arm.state.timer2.Read(addr); ok {
				return uint32(v)
			}
		}
		if addr == arm.mmap.APBDIV {
			return uint32(0)
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

func (arm *ARM) write32bit(addr uint32, val uint32, requiresAlignment bool) {
	if addr < arm.mmap.NullAccessBoundary {
		arm.nullAccess("Write 32bit", addr)
	}

	// check 32 bit alignment
	if (requiresAlignment || !arm.mmap.MisalignedAccesses) && !IsAlignedTo32bits(addr) {
		arm.misalignedAccess("Write 32bit", addr)
		if !arm.mmap.MisalignedAccesses {
			addr = AlignTo32bits(addr)
		}
	}

	mem, origin := arm.mem.MapAddress(addr, true, false)
	if mem == nil {
		if arm.mmap.HasMAM {
			if arm.state.mam.Write(addr, uint32(val)) {
				return
			}
		}
		if arm.mmap.HasRNG {
			if arm.state.rng.Write(addr, uint32(val)) {
				return
			}
		}
		if arm.mmap.HasT1 {
			if arm.state.timer.Write(addr, uint32(val)) {
				return
			}
		}
		if arm.mmap.HasTIM2 {
			if arm.state.timer2.Write(addr, uint32(val)) {
				return
			}
		}
		if addr == arm.mmap.APBDIV {
			return
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
	mem, origin := arm.mem.MapAddress(addr, false, false)
	addr -= origin
	if mem == nil || addr >= uint32(len(*mem)-3) {
		return 0, false
	}
	return arm.byteOrder.Uint32((*mem)[addr:]), true
}
