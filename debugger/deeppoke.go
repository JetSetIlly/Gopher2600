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
		err := dbg.searchDeepPoke(addr, value, newValue, valueMask)
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

// IsDeepPoking returns true if a deep poke search is in progress.
func (dbg *Debugger) IsDeepPoking() bool {
	return len(dbg.deepPoking) > 0
}

// searchDeepPoke looks for the literal value that is being poked into the quoted
// address. Once found the newValue is poked in appropriately.
//
// The valueMask is used during the rewind history search to help decide if the
// value being written matches the value we're looking for. See SearchMemoryWrite()
// and SearchRegisterWrite() in the rewind package.
func (dbg *Debugger) searchDeepPoke(addr uint16, value uint8, newValue uint8, valueMask uint8) (rerr error) {
	// dummy PC to do address indexing sums with
	dummyPC := registers.NewProgramCounter(0)

	// get current state and finish off the deeppoke process by resuming the
	// "real" emulation at that state
	resumeState := dbg.Rewind.GetCurrentState()

	// the address and state we'll be poking
	var pokeState *rewind.State
	var pokeAddr uint16

	// the most recent Poke RAM state which we can fall back on if we don't
	// find anything more solid
	var pokeRAMState *rewind.State
	var pokeRAMAddr uint16

	// how we poke memory differs depending on what we are poking. pokingRAM
	// notes whether pokeRAMState/pokeRAMaddr is being used
	var pokingRAM bool

	// we amost always have to call rewind.RunFromState() on the exit of searchDeepPoke()
	defer func() {
		if pokeState == nil {
			pokeState = resumeState
		}

		var pokeHook rewind.PokeHook

		if pokingRAM {
			pokeHook = func(s *rewind.State) error {
				return deepPoke(s.Mem, pokeAddr, newValue, valueMask)
			}
		} else {
			// if we're not poking RAM then we poke the state that we found.
			// this is particulatly important for cartridges with multiple
			// banks - we need to poke the bank we found.
			err := deepPoke(pokeState.Mem, pokeAddr, newValue, valueMask)
			if err != nil {
				rerr = err // superceding any existing error
				return
			}
		}

		fmt.Printf("running from (%s) to (%s)\n", pokeState.TV.String(), resumeState.TV.String())

		err := dbg.Rewind.RunFromState(pokeState, resumeState, pokeHook)
		if err != nil {
			rerr = err // superceding any existing error
		}
	}()

	// trace memory write as far back as we can
	searchState := resumeState
	depth := 0
	for depth < 4 {
		depth++

		var err error

		searchState, err = dbg.Rewind.SearchMemoryWrite(searchState, addr, value, valueMask)
		if err != nil {
			return nil
		}
		if searchState == nil {
			pokeAddr = pokeRAMAddr
			pokeState = pokeRAMState
			pokingRAM = true
			return nil
		}

		// basic disasm
		fmt.Printf("mem %d: %s [tv: %s]\n", depth, searchState.CPU.LastResult.String(), searchState.TV.String())

		// writes to memory always use a register as the source of the write
		reg := ""
		switch searchState.CPU.LastResult.Defn.Operator {
		case "STA":
			reg = "A"
		case "STX":
			reg = "X"
		case "STY":
			reg = "Y"
		default:
			return curated.Errorf("unexpected write sequence (%s)", searchState.CPU.LastResult.String())
		}

		searchState, err = dbg.Rewind.SearchRegisterWrite(searchState, reg, value, valueMask)
		if err != nil {
			return nil
		}
		if searchState == nil {
			pokeAddr = pokeRAMAddr
			pokeState = pokeRAMState
			pokingRAM = true
			return nil
		}

		// basic disasm
		fmt.Printf("reg %d: %s [tv: %s]\n", depth, searchState.CPU.LastResult.String(), searchState.TV.String())

		// writes to a register can happen from another register, and immediate
		// value, or an address in memory (most probably from the cartridge or
		// VCS RAM)
		switch searchState.CPU.LastResult.Defn.AddressingMode {
		case instructions.AbsoluteIndexedX:
			ma, area := memorymap.MapAddress(searchState.CPU.LastResult.InstructionData, false)
			if area == memorymap.Cartridge {
				dummyPC.Load(ma)
				dummyPC.Add(searchState.CPU.X.Address())
				pokeAddr = dummyPC.Address()
				pokeState = searchState
			}
			return nil
		case instructions.AbsoluteIndexedY:
			ma, area := memorymap.MapAddress(searchState.CPU.LastResult.InstructionData, false)
			if area == memorymap.Cartridge {
				dummyPC.Load(ma)
				dummyPC.Add(searchState.CPU.X.Address())
				pokeAddr = dummyPC.Address()
				pokeState = searchState
			}
			return nil
		case instructions.Immediate:
			ma, area := memorymap.MapAddress(searchState.CPU.LastResult.Address, false)
			if area == memorymap.Cartridge {
				dummyPC.Load(ma)
				dummyPC.Add(1)
				pokeAddr = dummyPC.Address()
				pokeState = searchState
			}
			return nil

		case instructions.ZeroPage:
			_, area := memorymap.MapAddress(searchState.CPU.LastResult.InstructionData, false)
			switch area {
			case memorymap.RAM:
				addr = searchState.CPU.LastResult.InstructionData
				pokeRAMAddr = addr
				pokeRAMState = searchState
			default:
				// zero-page access through a non-RAM zero page address seems unlikely
				return curated.Errorf("not tracing zero-page access to non-RAM (%s)", area)
			}
		case instructions.ZeroPageIndexedX:
			ma, area := memorymap.MapAddress(searchState.CPU.LastResult.InstructionData, false)
			switch area {
			case memorymap.RAM:
				dummyPC.Load(ma)
				dummyPC.Add(searchState.CPU.X.Address())
				addr = dummyPC.Address()
				pokeRAMAddr = addr
				pokeRAMState = searchState
			default:
				return curated.Errorf("not tracing zero-page access to non-RAM (%s)", area)
			}
		case instructions.ZeroPageIndexedY:
			ma, area := memorymap.MapAddress(searchState.CPU.LastResult.InstructionData, false)
			switch area {
			case memorymap.RAM:
				dummyPC.Load(ma)
				dummyPC.Add(searchState.CPU.Y.Address())
				addr = dummyPC.Address()
				pokeRAMAddr = addr
				pokeRAMState = searchState
			default:
				return curated.Errorf("not tracing zero-page access to non-RAM (%s)", area)
			}
		default:
			return curated.Errorf("unsupported addressing mode (%s)", searchState.CPU.LastResult.String())
		}
	}

	return curated.Errorf("poking too deep")
}

// deepPoke changes the bits at the address according the value of valueMask.
// Change all bits with a mask of 0xff.
func deepPoke(mem *memory.Memory, addr uint16, newValue uint8, valueMask uint8) error {
	v, err := mem.Peek(addr)
	if err != nil {
		return err
	}
	v = (v & ^valueMask) | (newValue & valueMask)
	fmt.Printf("deep poking %02x into %04x\n", v, addr)
	return mem.Poke(addr, v)
}
