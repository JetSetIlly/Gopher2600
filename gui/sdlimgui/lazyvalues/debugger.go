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

// LazyDebugger lazily accesses Debugger information
type LazyDebugger struct {
	val *Lazy

	atomicQuantum    atomic.Value // debugger.QuantumMode
	atomicLastResult atomic.Value // disassembly.Entry
	Quantum          debugger.QuantumMode

	// a LastResult value is also part of the reflection structure but it's
	// more convenient to get it direcetly, in addition to reflection.
	LastResult disassembly.Entry
}

func newLazyDebugger(val *Lazy) *LazyDebugger {
	lz := &LazyDebugger{val: val}
	return lz
}

func (lz *LazyDebugger) update() {
	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicQuantum.Store(lz.val.Dbg.GetQuantum())
		lz.atomicLastResult.Store(lz.val.Dbg.GetLastResult())
	})
	lz.Quantum, _ = lz.atomicQuantum.Load().(debugger.QuantumMode)

	if lz.atomicLastResult.Load() != nil {
		lz.LastResult = lz.atomicLastResult.Load().(disassembly.Entry)
	}
}
