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

package lazyvalues

import (
	"sync/atomic"

	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/disassembly"
)

// LazyDebugger lazily accesses Debugger information.
type LazyDebugger struct {
	val *LazyValues

	quantum         atomic.Value // debugger.QuantumMode
	liveDisasmEntry atomic.Value // disassembly.Entry
	breakpoints     atomic.Value // debugger.BreakpointsQuery
	hasChanged      atomic.Value // bool
	state           atomic.Value // govern.State

	Quantum         debugger.Quantum
	LiveDisasmEntry disassembly.Entry
	Breakpoints     debugger.LazyBreakpointsQuery
	HasChanged      bool

	// the govern.State below is taken at the same time as the reset of the
	// lazy values. this value should be used in preference to the live
	// govern.State() value (which is safe to obtain outside of the lazy
	// system) when synchronisation is important
	State govern.State
}

func newLazyDebugger(val *LazyValues) *LazyDebugger {
	lz := &LazyDebugger{val: val}
	lz.hasChanged.Store(false)
	return lz
}

func (lz *LazyDebugger) push() {
	lz.quantum.Store(lz.val.dbg.LazyGetQuantum())
	lz.liveDisasmEntry.Store(lz.val.dbg.LazyGetLiveDisasmEntry())
	lz.breakpoints.Store(lz.val.dbg.LazyQueryBreakpoints())

	// because the push() and update() pair don't interlock exactly, the
	// hasChanged field must be latched. unlatching is performed in the
	// update() function
	if !lz.hasChanged.Load().(bool) {
		lz.hasChanged.Store(lz.val.dbg.LazyHasChanged())
	}

	lz.state.Store(lz.val.dbg.State())
}

func (lz *LazyDebugger) update() {
	lz.Quantum, _ = lz.quantum.Load().(debugger.Quantum)

	if lz.liveDisasmEntry.Load() != nil {
		lz.LiveDisasmEntry = lz.liveDisasmEntry.Load().(disassembly.Entry)
	}

	lz.Breakpoints, _ = lz.breakpoints.Load().(debugger.LazyBreakpointsQuery)

	// load current hasChanged value and unlatch (see push() function)
	lz.HasChanged = lz.hasChanged.Load().(bool)
	lz.hasChanged.Store(false)
	lz.State, _ = lz.state.Load().(govern.State)
}
