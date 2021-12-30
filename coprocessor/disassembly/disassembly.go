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

package disassembly

import (
	"sort"
	"sync"

	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

// CoProcessor is used to handle the disassembly of instructions from an
// attached cartridge that contains a coprocessor.
type CoProcessor struct {
	crit sync.Mutex
	vcs  *hardware.VCS

	enabled bool

	cart mapper.CartCoProcBus

	disasm     map[string]mapper.CartCoProcDisasmEntry
	disasmKeys []string // sorted keys into the disasm map

	lastExecution        []mapper.CartCoProcDisasmEntry
	lastExecutionSummary mapper.CartCoProcDisasmSummary

	lastStart coords.TelevisionCoords
}

// NewCoProcessorDisasm returns a new Coprocessor instance if cartridge implements the
// coprocessor bus.
func NewCoProcessorDisasm(vcs *hardware.VCS, cart *cartridge.Cartridge) *CoProcessor {
	cop := &CoProcessor{
		vcs:           vcs,
		lastExecution: make([]mapper.CartCoProcDisasmEntry, 0, 1024),
	}

	cop.cart = cart.GetCoProcBus()
	if cop.cart == nil {
		return nil
	}

	cop.disasm = make(map[string]mapper.CartCoProcDisasmEntry)
	cop.disasmKeys = make([]string, 0, 1024)

	cop.Enable(false)

	return cop
}

// IsEnabled returns true if coprocessor disassembly is currently active.
func (cop *CoProcessor) IsEnabled() bool {
	cop.crit.Lock()
	defer cop.crit.Unlock()
	return cop.enabled
}

// Enable or disable coprocessor disassembly. We retain the disassembly
// (including last execution) already gathered but the LastExecution field is
// cleared on disable. The general disassembly is maintained.
func (cop *CoProcessor) Enable(enable bool) {
	cop.crit.Lock()
	defer cop.crit.Unlock()

	cop.enabled = enable
	if cop.enabled {
		cop.cart.SetDisassembler(cop)
	} else {
		cop.cart.SetDisassembler(nil)
		cop.lastExecution = cop.lastExecution[:0]
	}
}

// Start implements the CartCoProcDisassembler interface.
func (cop *CoProcessor) Start() {
	cop.crit.Lock()
	defer cop.crit.Unlock()

	if cop.enabled {
		// add one clock to frame/scanline/clock values. the Reset() function will
		// have been called on the last CPU cycle of the instruction that triggers
		// the coprocessor reset. the TV will not have moved onto the beginning of
		// the next instruction yet so we must figure it out here
		cop.lastStart = cop.vcs.TV.AdjCoords(television.AdjCPUCycle, 1)
	}

	cop.lastExecution = cop.lastExecution[:0]
}

// Step implements the CartCoProcDisassembler interface.
func (cop *CoProcessor) Step(entry mapper.CartCoProcDisasmEntry) {
	cop.crit.Lock()
	defer cop.crit.Unlock()

	// check that coprocessor disassmebler hasn't been disabled in the period
	// while we were waiting for the critical section lock
	if !cop.enabled {
		return
	}

	cop.lastExecution = append(cop.lastExecution, entry)
}

// End implements the CartCoProcDisassembler interface.
func (cop *CoProcessor) End(summary mapper.CartCoProcDisasmSummary) {
	cop.crit.Lock()
	defer cop.crit.Unlock()

	cop.lastExecutionSummary = summary

	for _, entry := range cop.lastExecution {
		key := entry.Key()
		if _, ok := cop.disasm[key]; !ok {
			cop.disasmKeys = append(cop.disasmKeys, key)
		}
		cop.disasm[key] = entry
	}

	sort.Strings(cop.disasmKeys)
}
