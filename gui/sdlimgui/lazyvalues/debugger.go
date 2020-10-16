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
	"github.com/jetsetilly/gopher2600/disassembly"
)

// LazyDebugger lazily accesses Debugger information.
type LazyDebugger struct {
	val *LazyValues

	quantum    atomic.Value // debugger.QuantumMode
	lastResult atomic.Value // disassembly.Entry

	Quantum    debugger.QuantumMode
	LastResult disassembly.Entry
}

func newLazyDebugger(val *LazyValues) *LazyDebugger {
	lz := &LazyDebugger{val: val}
	return lz
}

func (lz *LazyDebugger) push() {
	lz.quantum.Store(lz.val.Dbg.GetQuantum())
	lz.lastResult.Store(lz.val.Dbg.GetLastResult())
}

func (lz *LazyDebugger) update() {
	lz.Quantum, _ = lz.quantum.Load().(debugger.QuantumMode)
	if lz.lastResult.Load() != nil {
		lz.LastResult = lz.lastResult.Load().(disassembly.Entry)
	}
}
