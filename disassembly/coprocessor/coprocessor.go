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
	"sort"
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

	entries     map[string]mapper.CartCoProcDisasmEntry
	entriesKeys []string
}

type LastExecutionDetails struct {
	// values at beginning of execution
	Frame    int
	Scanline int
	Clock    int

	// values at end of execution
	Summary mapper.CartCoProcExecutionSummary
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
	cop.entries = make(map[string]mapper.CartCoProcDisasmEntry)
	cop.entriesKeys = make([]string, 0, 1024)
	cpd.SetDisassembler(cop)
	return cop
}

// Start implements the CartCoProcDisassembler interface.
func (cop *Coprocessor) Start() {
	cop.crit.Lock()
	defer cop.crit.Unlock()

	cop.lastExecution = cop.lastExecution[:0]

	// add one clock to frame/scanline/clock values. the Reset() function will
	// have been called on the last CPU cycle of the instruction that triggers
	// the coprocessor reset. the TV will not have moved onto the beginning of
	// the next instruction yet so we must figure it out here
	fn, sl, cl, _ := cop.vcs.TV.ReqAdjust(signal.AdjCPUCycle, 1, false)
	cop.lastExecutionDetails.Frame = fn
	cop.lastExecutionDetails.Scanline = sl
	cop.lastExecutionDetails.Clock = cl
}

// Step implements the CartCoProcDisassembler interface.
func (cop *Coprocessor) Step(entry mapper.CartCoProcDisasmEntry) {
	cop.crit.Lock()
	defer cop.crit.Unlock()
	cop.lastExecution = append(cop.lastExecution, entry)

	if _, ok := cop.entries[entry.Address]; !ok {
		cop.entriesKeys = append(cop.entriesKeys, entry.Address)
		sort.Strings(cop.entriesKeys)
	}
	cop.entries[entry.Address] = entry
}

// End implements the CartCoProcDisassembler interface.
func (cop *Coprocessor) End(summary mapper.CartCoProcExecutionSummary) {
	cop.crit.Lock()
	defer cop.crit.Unlock()
	cop.lastExecutionDetails.Summary = summary
}
