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

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/cpu/registers"
	"github.com/jetsetilly/gopher2600/hardware/memory"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/rewind"
)

// IsDeepPoking returns true if a deep poke search is in progress.
func (dbg *Debugger) IsDeepPoking() bool {
	return len(dbg.deepPoking) > 0
}

// PushDeepPoke schedules a deep poke search for the specificed value in the
// address, replacing it with newValue if found. The valueMask will be applied
// to the value for matching and for setting the newValue - unset bits in the
// mask will preserve the corresponding bits in the found value.
func (dbg *Debugger) PushDeepPoke(addr uint16, value uint8, newValue uint8, valueMask uint8) bool {
	// try pushing to the deepPoking channel.
	//
	// if we cannot then that means a deep-poke search is currently taking place and we
	// return false to indicate that the request has not taken place yet.
	select {
	case dbg.deepPoking <- true:
	default:
		logger.Logf("deep-poke", "delaying poke of %04x", addr)
		return false
	}

	dbg.PushRawEventImm(func() {
		err := dbg.doDeepPoke(addr, value, newValue, valueMask)
		if err != nil {
			logger.Logf("deep-poke", "%v", err)
		}

		// unblock deepPoking channel
		select {
		case <-dbg.deepPoking:
		default:
		}
	})

	return true
}

// returned by searchDeepPoke() for convenience.
type deepPoking struct {
	state *rewind.State
	addr  uint16
	area  memorymap.Area
}

func (dbg *Debugger) doDeepPoke(addr uint16, value uint8, newValue uint8, valueMask uint8) error {
	// get current state and finish off the deep-poke process by resuming the
	// "real" emulation at that state
	currentState := dbg.Rewind.GetCurrentState()

	poking, err := dbg.searchDeepPoke(currentState, addr, value, valueMask)
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
			return deepPoke(s.Mem, poking.addr, newValue, valueMask)
		}

		// run from found poking.state to the "current state" in the real emulation
		err = dbg.Rewind.RunFromState(poking.state, currentState, pokeHook)
		if err != nil {
			return err
		}
	case memorymap.Cartridge:
		// if we're not poking RAM then we poke the state that we found
		// immediately. this is particulatly important for cartridges with
		// multiple banks because we need to poke the bank we found.
		err := deepPoke(poking.state.Mem, poking.addr, newValue, valueMask)
		if err != nil {
			return err
		}

		// run from found poking.state to the "current state" in the real emulation
		err = dbg.Rewind.RunFromState(poking.state, currentState, nil)
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

	fmt.Println("----")
	fmt.Printf("searching for %02x write to %04x (mask=%02x)\n", value, searchAddr, valueMask)

	// trace memory write as far back as we can
	for depth := 0; depth < maxSearchDepth; depth++ {
		searchState, err = dbg.Rewind.SearchMemoryWrite(searchState, searchAddr, value, valueMask)
		if err != nil {
			return deepPoking{}, err
		}
		if searchState == nil {
			return poking, nil
		}

		fmt.Println("(mem)", searchState.CPU.LastResult.String())

		// writes to memory always use a register as the source of the write
		var reg rune
		switch searchState.CPU.LastResult.Defn.Operator {
		case "STA":
			reg = 'A'
		case "STX":
			reg = 'X'
		case "STY":
			reg = 'Y'
		default:
			return deepPoking{}, curated.Errorf("unexpected write sequence (%s)", searchState.CPU.LastResult.String())
		}

		searchState, err = dbg.Rewind.SearchRegisterWrite(searchState, reg, value, valueMask)
		if err != nil {
			return deepPoking{}, err
		}
		if searchState == nil {
			return poking, nil
		}

		// writes to a register can happen from another register, and immediate
		// value, or an address in memory (most probably from the cartridge or
		// VCS RAM)
		switch searchState.CPU.LastResult.Defn.AddressingMode {
		case instructions.Immediate:
			ma, area := memorymap.MapAddress(searchState.CPU.LastResult.Address, false)
			switch area {
			case memorymap.Cartridge:
				fmt.Println("(reg)", searchState.CPU.LastResult.String())
				pc := registers.NewProgramCounter(ma)
				pc.Add(1)
				poking.addr = pc.Address()
				poking.state = searchState
				poking.area = area
				return poking, nil
			default:
				return deepPoking{}, curated.Errorf("not deep-poking through non-RAM/Cartridge space (%s)", area)
			}
		case instructions.AbsoluteIndexedX:
			ma, area := memorymap.MapAddress(searchState.CPU.LastResult.InstructionData, false)
			switch area {
			case memorymap.Cartridge:
				fmt.Println("(reg)", searchState.CPU.LastResult.String(), searchState.CPU.X.Value())
				pc := registers.NewProgramCounter(ma)
				pc.Add(searchState.CPU.X.Address())
				poking.addr = pc.Address()
				poking.state = searchState
				poking.area = area
				return poking, nil
			case memorymap.RAM:
				fmt.Println("(reg)", searchState.CPU.LastResult.String(), searchState.CPU.X.Value())
				pc := registers.NewProgramCounter(ma)
				pc.Add(searchState.CPU.X.Address())
				poking.addr = pc.Address()
				poking.state = searchState
				poking.area = area
			default:
				return deepPoking{}, fmt.Errorf("not deep-poking through non-RAM/Cartridge space (%s)", area)
			}
		case instructions.AbsoluteIndexedY:
			ma, area := memorymap.MapAddress(searchState.CPU.LastResult.InstructionData, false)
			switch area {
			case memorymap.Cartridge:
				fmt.Println("(reg)", searchState.CPU.LastResult.String(), searchState.CPU.Y.Value())
				pc := registers.NewProgramCounter(ma)
				pc.Add(searchState.CPU.Y.Address())
				poking.addr = pc.Address()
				poking.state = searchState
				poking.area = area
				return poking, nil
			case memorymap.RAM:
				fmt.Println("(reg)", searchState.CPU.LastResult.String(), searchState.CPU.Y.Value())
				pc := registers.NewProgramCounter(ma)
				pc.Add(searchState.CPU.Y.Address())
				poking.addr = pc.Address()
				poking.state = searchState
				poking.area = area
			default:
				return deepPoking{}, curated.Errorf("not deep-poking through non-RAM/Cartridge space (%s)", area)
			}
		case instructions.ZeroPage:
			ma, area := memorymap.MapAddress(searchState.CPU.LastResult.InstructionData, false)
			switch area {
			case memorymap.RAM:
				fmt.Println("(reg)", searchState.CPU.LastResult.String())
				poking.addr = ma
				poking.state = searchState
				poking.area = area

				// update the search address and continue with the search
				searchAddr = poking.addr
			default:
				return deepPoking{}, curated.Errorf("not deep-poking through non-RAM space (%s)", area)
			}
		case instructions.ZeroPageIndexedX:
			ma, area := memorymap.MapAddress(searchState.CPU.LastResult.InstructionData, false)
			switch area {
			case memorymap.RAM:
				fmt.Println("(reg)", searchState.CPU.LastResult.String(), searchState.CPU.X.Value())
				pc := registers.NewProgramCounter(ma)
				pc.Add(searchState.CPU.X.Address())
				poking.addr = pc.Address()
				poking.state = searchState
				poking.area = area

				// update the search address and continue with the search
				searchAddr = poking.addr
			default:
				return deepPoking{}, curated.Errorf("not deep-poking through non-RAM space (%s)", area)
			}
		case instructions.ZeroPageIndexedY:
			ma, area := memorymap.MapAddress(searchState.CPU.LastResult.InstructionData, false)
			switch area {
			case memorymap.RAM:
				fmt.Println("(reg)", searchState.CPU.LastResult.String(), searchState.CPU.Y.Value())
				pc := registers.NewProgramCounter(ma)
				pc.Add(searchState.CPU.Y.Address())
				poking.addr = pc.Address()
				poking.state = searchState
				poking.area = area

				// update the search address and continue with the search
				searchAddr = poking.addr
			default:
				return deepPoking{}, curated.Errorf("not deep-poking through non-RAM space (%s)", area)
			}
		default:
			return deepPoking{}, curated.Errorf("unsupported addressing mode (%s)", searchState.CPU.LastResult.String())
		}
	}

	return deepPoking{}, curated.Errorf("deep-poking too deep")
}

// deepPoke changes the bits at the address according the value of valueMask.
func deepPoke(mem *memory.Memory, addr uint16, value uint8, valueMask uint8) error {
	v, err := mem.Peek(addr)
	if err != nil {
		return err
	}
	v = (v & (valueMask ^ 0xff)) | (value & valueMask)
	fmt.Printf("deep-poke %04x <- %02x\n", addr, value)
	return mem.Poke(addr, v)
}
