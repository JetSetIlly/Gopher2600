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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

// Iterate facilitates traversal over the disasm of the the last execution of
// the coprocessor.
type Iterate struct {
	// copy of the lastExecution because it can change in the emulation
	// goroutine us at any time
	disasm []mapper.CartCoProcDisasmEntry

	// information about the last execution (eg. screen coordinates)
	Summary mapper.CartCoProcDisasmSummary

	// the coordinates of the most recent execution in the disassembly
	LastStart coords.TelevisionCoords

	// number of entries in the iterations
	Count int

	// the next entry to be returned by the Next() function
	idx int
}

type IterationType int

const (
	LastExecution IterationType = iota
	Disassembly
)

// NewIteration is the preferred method if initialistation for the Iterate
// type.
func (cop *CoProcessor) NewIteration(iterationType IterationType) *Iterate {
	cop.crit.Lock()
	defer cop.crit.Unlock()

	itr := &Iterate{}

	if iterationType == LastExecution {
		itr.disasm = make([]mapper.CartCoProcDisasmEntry, len(cop.lastExecution))
		copy(itr.disasm, cop.lastExecution)
	} else if iterationType == Disassembly {
		itr.disasm = make([]mapper.CartCoProcDisasmEntry, 0, len(cop.disasm))
		for _, k := range cop.disasmKeys {
			itr.disasm = append(itr.disasm, cop.disasm[k])
		}
	}
	itr.Count = len(itr.disasm)
	itr.Summary = cop.lastExecutionSummary
	itr.LastStart = cop.lastStart

	return itr
}

// Start new iterations.
func (itr *Iterate) Start() (*mapper.CartCoProcDisasmEntry, bool) {
	itr.idx = -1
	return itr.next()
}

// Return the next entry in the iteration.
func (itr *Iterate) Next() (*mapper.CartCoProcDisasmEntry, bool) {
	return itr.next()
}

// Skip the next N entries of the entry and return that entry.
func (itr *Iterate) SkipNext(n int) (*mapper.CartCoProcDisasmEntry, bool) {
	itr.idx += n - 1
	return itr.next()
}

func (itr *Iterate) next() (*mapper.CartCoProcDisasmEntry, bool) {
	if itr.idx+1 >= itr.Count {
		return nil, false
	}

	itr.idx++
	return &itr.disasm[itr.idx], true
}
