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

package debugger

import (
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/rewind"
)

// DeepPoke looks for the literal value that is being poked into the quoted
// address. Once found the newValue is poked in appropriately.
//
// The valueMask is used during the rewind history search to help decide if the
// value being written matches the value we're looking for. See SearchMemoryWrite()
// and SearchRegisterWrite() in the rewind package.
func (dbg *Debugger) DeepPoke(addr uint16, value uint8, newValue uint8, valueMask uint8) error {
	var err error

	// get current state and finish off the deeppoke process by resuming the
	// "real" emulation at that state
	currState := dbg.Rewind.GetCurrentState()
	res := currState

	defer func() {
		dbg.Rewind.RunFromState(res, currState) // ignoring error
	}()

	depth := 0
	for depth < 4 {
		depth++

		res, err = dbg.Rewind.SearchMemoryWrite(res, addr, value, valueMask)
		if err != nil {
			return curated.Errorf("deep-poke: %v", err)
		}
		if res == nil {
			return curated.Errorf("deep-poke: cannot find write to memory")
		}

		// writes to memory always use a register as the source of the write
		reg := ""
		switch res.CPU.LastResult.Defn.Operator {
		case "STA":
			reg = "A"
		case "STX":
			reg = "X"
		case "STY":
			reg = "Y"
		default:
			return curated.Errorf("deep-poke: unexpected write sequence (%s)", res.CPU.LastResult.String())
		}

		res, err = dbg.Rewind.SearchRegisterWrite(res, reg, value, valueMask)
		if err != nil {
			return curated.Errorf("deep-poke: %v", err)
		}
		if res == nil {
			return curated.Errorf("deep-poke: cannot find write to register")
		}

		// writes to a register can happen from another register, and immediate
		// value, or an address in memory (most probably from the cartridge or
		// VCS RAM)
		switch res.CPU.LastResult.Defn.AddressingMode {
		case instructions.AbsoluteIndexedX:
			ma, area := memorymap.MapAddress(res.CPU.LastResult.InstructionData, false)
			if area == memorymap.Cartridge {
				ma += uint16(res.CPU.X.Value())
				err = deepPoke(res, ma, newValue, valueMask)
				if err != nil {
					return curated.Errorf("deep-poke: %v", err)
				}
			}
			return nil
		case instructions.AbsoluteIndexedY:
			ma, area := memorymap.MapAddress(res.CPU.LastResult.InstructionData, false)
			if area == memorymap.Cartridge {
				ma += uint16(res.CPU.Y.Value())
				err = deepPoke(res, ma, newValue, valueMask)
				if err != nil {
					return curated.Errorf("deep-poke: %v", err)
				}
			}
			return nil
		case instructions.Immediate:
			ma, area := memorymap.MapAddress(res.CPU.LastResult.Address, false)
			if area == memorymap.Cartridge {
				err = deepPoke(res, ma+1, newValue, valueMask)
				if err != nil {
					return curated.Errorf("deep-poke: %v", err)
				}
			}
			return nil
		case instructions.ZeroPage:
			addr = res.CPU.LastResult.InstructionData
		default:
			return curated.Errorf("deep-poke: unsupported addressing mode (%s)", res.CPU.LastResult.String())
		}
	}

	return curated.Errorf("deep-poke: poking too deep")
}

// deepPoke changes the bits at the address according the value of valueMask.
// Change all bits with a mask of 0xff.
func deepPoke(res *rewind.State, addr uint16, newValue uint8, valueMask uint8) error {
	v, err := res.Mem.Peek(addr)
	if err != nil {
		return err
	}
	v = (v & ^valueMask) | (newValue & valueMask)
	return res.Mem.Poke(addr, v)
}
