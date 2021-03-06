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

package coprocessor

import (
	"sync"

	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
)

// Coprocessor is used to handle the disassembly of instructions from an
// attached cartridge that contains a coprocessor.
type Coprocessor struct {
	crit sync.Mutex
	vcs  *hardware.VCS

	lastExecution        []mapper.CartCoProcDisasmEntry
	lastExecutionDetails LastExecutionDetails
}

type LastExecutionDetails struct {
	Frame    int
	Scanline int
	Clock    int
}

// Add returns a new Coprocessor instance if cartridge implements the
// coprocessor bus.
func Add(vcs *hardware.VCS, cart *cartridge.Cartridge) *Coprocessor {
	cpd := cart.GetCoProcBus()
	if cpd == nil {
		return nil
	}

	cop := &Coprocessor{
		vcs:           vcs,
		lastExecution: make([]mapper.CartCoProcDisasmEntry, 0, 1024),
	}
	cpd.SetDisassembler(cop)
	return cop
}

// Reset implements the CartCoProcDisassembler interface.
func (cop *Coprocessor) Reset() {
	cop.crit.Lock()
	defer cop.crit.Unlock()

	// add one clock to frame/scanline/clock values. the Reset() function will
	// have been called on the last CPU cycle of the instruction that triggers
	// the coprocessor reset. the TV will not have moved onto the beginning of
	// the next instruction yet so we must figure it out here
	fn, sl, cl, _ := cop.vcs.TV.ReqAdjust(signal.AdjCPUCycle, 1, false)
	cop.lastExecutionDetails.Frame = fn
	cop.lastExecutionDetails.Scanline = sl
	cop.lastExecutionDetails.Clock = cl

	cop.lastExecution = cop.lastExecution[:0]
}

// Instruction implements the CartCoProcDisassembler interface.
func (cop *Coprocessor) Instruction(entry mapper.CartCoProcDisasmEntry) {
	cop.crit.Lock()
	defer cop.crit.Unlock()
	cop.lastExecution = append(cop.lastExecution, entry)
}
