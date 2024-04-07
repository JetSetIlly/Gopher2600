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
	"fmt"

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/cpu/registers"
	"github.com/jetsetilly/gopher2600/hardware/memory"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/rewind"
)

// PushDeepPoke schedules a deep poke search for the specificed value in the
// address, replacing it with newValue if found. The valueMask will be applied
// to the value for matching and for setting the newValue - unset bits in the
// mask will preserve the corresponding bits in the found value.
func (dbg *Debugger) PushDeepPoke(addr uint16, value uint8, newValue uint8, valueMask uint8, done func()) bool {
	// get current state to use as the resume state after the deeppoke and rewind recovery
	searchState := dbg.Rewind.GetCurrentState()

	doDeepPoke := func() error {
		err := dbg.doDeepPoke(searchState, addr, value, newValue, valueMask)
		if err != nil {
			logger.Logf("deeppoke", "%v", err)
		}
		if done != nil {
			done()
		}
		return nil
	}

	dbg.PushFunctionImmediate(func() {
		dbg.setState(govern.Rewinding, govern.Normal)
		dbg.unwindLoop(doDeepPoke)
	})

	return true
}

// returned by searchDeepPoke() for convenience.
type deepPoking struct {
	state *rewind.State
	addr  uint16
	area  memorymap.Area
}

func (dbg *Debugger) doDeepPoke(searchState *rewind.State, addr uint16, value uint8, newValue uint8, valueMask uint8) error {
	poking, err := dbg.searchDeepPoke(searchState, addr, value, valueMask)
	if err != nil {
		return err
	}

	if poking.state == nil {
		return nil
	}

	switch poking.area {
	case memorymap.RAM:
		// poking RAM requires a rewind.PokeHook. this is so we can poke
		// the new value at the correct point rather than from the resume
		// state.
		pokeHook := func(s *rewind.State) error {
			return deepPoke(s.VCS.Mem, poking, newValue, valueMask)
		}

		// run from found poking.state to the "current state" in the real emulation
		err = dbg.Rewind.RunPoke(poking.state, searchState, pokeHook)
		if err != nil {
			return err
		}
	case memorymap.Cartridge:
		// if we're not poking RAM then we poke the state that we found
		// immediately. this is particulatly important for cartridges with
		// multiple banks because we need to poke the bank we found.
		err := deepPoke(poking.state.VCS.Mem, poking, newValue, valueMask)
		if err != nil {
			return err
		}

		// run from found poking.state to the "current state" in the real emulation
		err = dbg.Rewind.RunPoke(poking.state, searchState, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// searchDeepPoke looks for the literal value that is being poked into the quoted
// address. Once found the newValue is poked in appropriately.
//
// The valueMask is used during the rewind history search to help decide if the
// value being written matches the value we're looking for. See SearchMemoryWrite()
// and SearchRegisterWrite() in the rewind package.
func (dbg *Debugger) searchDeepPoke(searchState *rewind.State, searchAddr uint16, value uint8, valueMask uint8) (deepPoking, error) {
	var poking deepPoking
	var err error

	const maxSearchDepth = 10

	// trace memory write as far back as we can
	for depth := 0; depth < maxSearchDepth; depth++ {
		searchState, err = dbg.Rewind.SearchMemoryWrite(searchState, searchAddr, value, valueMask)
		if err != nil {
			return deepPoking{}, err
		}
		if searchState == nil {
			return poking, nil
		}

		// writes to memory always use a register as the source of the write
		var reg rune
		switch searchState.VCS.CPU.LastResult.Defn.Operator {
		case instructions.Sta:
			reg = 'A'
		case instructions.Stx:
			reg = 'X'
		case instructions.Sty:
			reg = 'Y'
		default:
			return deepPoking{}, fmt.Errorf("unexpected write sequence (%s)", searchState.VCS.CPU.LastResult.String())
		}

		searchState, err = dbg.Rewind.SearchRegisterWrite(searchState, reg, value, valueMask)
		if err != nil {
			return deepPoking{}, err
		}
		if searchState == nil {
			return poking, nil
		}

		if searchState.VCS.CPU.LastResult.Defn == nil {
			return deepPoking{}, fmt.Errorf("unexpected CPU result with a nil definition")
		}

		// writes to a register can happen from another register, and immediate
		// value, or an address in memory (most probably from the cartridge or
		// VCS RAM)
		switch searchState.VCS.CPU.LastResult.Defn.AddressingMode {
		case instructions.Immediate:
			ma, area := memorymap.MapAddress(searchState.VCS.CPU.LastResult.Address, false)
			switch area {
			case memorymap.Cartridge:
				pc := registers.NewProgramCounter(ma)
				pc.Add(1)
				poking.addr = pc.Address()
				poking.state = searchState
				poking.area = area
				return poking, nil
			default:
				return deepPoking{}, fmt.Errorf("not deeppoking through non-RAM/Cartridge space (%s)", area)
			}
		case instructions.AbsoluteIndexedX:
			ma, area := memorymap.MapAddress(searchState.VCS.CPU.LastResult.InstructionData, false)
			switch area {
			case memorymap.Cartridge:
				pc := registers.NewProgramCounter(ma)
				pc.Add(searchState.VCS.CPU.X.Address())
				poking.addr = pc.Address()
				poking.state = searchState
				poking.area = area
				return poking, nil
			case memorymap.RAM:
				pc := registers.NewProgramCounter(ma)
				pc.Add(searchState.VCS.CPU.X.Address())
				poking.addr = pc.Address()
				poking.state = searchState
				poking.area = area
			default:
				return deepPoking{}, fmt.Errorf("not deeppoking through non-RAM/Cartridge space (%s)", area)
			}
		case instructions.AbsoluteIndexedY:
			ma, area := memorymap.MapAddress(searchState.VCS.CPU.LastResult.InstructionData, false)
			switch area {
			case memorymap.Cartridge:
				pc := registers.NewProgramCounter(ma)
				pc.Add(searchState.VCS.CPU.Y.Address())
				poking.addr = pc.Address()
				poking.state = searchState
				poking.area = area
				return poking, nil
			case memorymap.RAM:
				pc := registers.NewProgramCounter(ma)
				pc.Add(searchState.VCS.CPU.Y.Address())
				poking.addr = pc.Address()
				poking.state = searchState
				poking.area = area
			default:
				return deepPoking{}, fmt.Errorf("not deeppoking through non-RAM/Cartridge space (%s)", area)
			}
		case instructions.ZeroPage:
			ma, area := memorymap.MapAddress(searchState.VCS.CPU.LastResult.InstructionData, false)
			switch area {
			case memorymap.RAM:
				poking.addr = ma
				poking.state = searchState
				poking.area = area

				// update the search address and continue with the search
				searchAddr = poking.addr
			default:
				return deepPoking{}, fmt.Errorf("not deeppoking through non-RAM space (%s)", area)
			}
		case instructions.ZeroPageIndexedX:
			ma, area := memorymap.MapAddress(searchState.VCS.CPU.LastResult.InstructionData, false)
			switch area {
			case memorymap.RAM:
				pc := registers.NewProgramCounter(ma)
				pc.Add(searchState.VCS.CPU.X.Address())
				poking.addr = pc.Address()
				poking.state = searchState
				poking.area = area

				// update the search address and continue with the search
				searchAddr = poking.addr
			default:
				return deepPoking{}, fmt.Errorf("not deeppoking through non-RAM space (%s)", area)
			}
		case instructions.ZeroPageIndexedY:
			ma, area := memorymap.MapAddress(searchState.VCS.CPU.LastResult.InstructionData, false)
			switch area {
			case memorymap.RAM:
				pc := registers.NewProgramCounter(ma)
				pc.Add(searchState.VCS.CPU.Y.Address())
				poking.addr = pc.Address()
				poking.state = searchState
				poking.area = area

				// update the search address and continue with the search
				searchAddr = poking.addr
			default:
				return deepPoking{}, fmt.Errorf("not deeppoking through non-RAM space (%s)", area)
			}
		case instructions.IndirectIndexed:
			pc := registers.NewProgramCounter(searchState.VCS.CPU.LastResult.InstructionData)
			lo, err := searchState.VCS.Mem.Read(pc.Address())
			if err != nil {
				return deepPoking{}, err
			}
			pc.Add(1)
			hi, err := searchState.VCS.Mem.Read(pc.Address())
			if err != nil {
				return deepPoking{}, err
			}
			pc.Load((uint16(hi) << 8) | uint16(lo))
			pc.Add(searchState.VCS.CPU.Y.Address())

			ma, area := memorymap.MapAddress(pc.Address(), false)
			switch area {
			case memorymap.Cartridge:
				poking.addr = ma
				poking.state = searchState
				poking.area = area
				return poking, nil
			case memorymap.RAM:
				poking.addr = ma
				poking.state = searchState
				poking.area = area
			default:
				return deepPoking{}, fmt.Errorf("not deeppoking through non-RAM/Cartridge space (%s)", area)
			}

			fallthrough
		default:
			return deepPoking{}, fmt.Errorf("unsupported addressing mode (%s)", searchState.VCS.CPU.LastResult.String())
		}
	}

	return deepPoking{}, fmt.Errorf("deeppoking too deep")
}

// deepPoke changes the bits at the address according the value of valueMask.
func deepPoke(mem *memory.Memory, poke deepPoking, value uint8, valueMask uint8) error {
	v, err := mem.Peek(poke.addr)
	if err != nil {
		return err
	}
	v = (v & (valueMask ^ 0xff)) | (value & valueMask)
	logger.Log("deeppoke", fmt.Sprintf("changing %#04x (%s) to %#02x", poke.addr, poke.area, value))
	return mem.Poke(poke.addr, v)
}
