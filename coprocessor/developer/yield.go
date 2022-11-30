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

package developer

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

// YieldState records the most recent yield.
type YieldState struct {
	InstructionPC   uint32
	InstructionLine *SourceLine
	Reason          mapper.YieldReason

	LocalVariables []*SourceVariable
}

// Cmp returns true if two YieldStates are equal.
func (y *YieldState) Cmp(w *YieldState) bool {
	return y.InstructionPC == w.InstructionPC && y.Reason == w.Reason
}

// OnYield implements the mapper.CartCoProcDeveloper interface.
func (dev *Developer) OnYield(instructionPC uint32, reason mapper.YieldReason) {
	var ln *SourceLine
	var locals []*SourceVariable

	// using BorrowSource because we want to make sure the source lock is
	// released if there is an error and the code panics
	dev.BorrowSource(func(src *Source) {
		// make sure that src is valid
		if src == nil {
			return
		}

		ln = src.FindSourceLine(instructionPC)
		if ln == nil {
			ln = createStubLine(nil)
		}

		// log a bug for any of these reasons
		switch reason {
		case mapper.YieldMemoryAccessError:
			fallthrough
		case mapper.YieldExecutionError:
			fallthrough
		case mapper.YieldUnimplementedFeature:
			fallthrough
		case mapper.YieldUndefinedBehaviour:
			if src != nil {
				if ln != nil {
					ln.Bug = true
				}
			}
		}

		// match local variables for any reason other than VCS synchronisation
		//
		// yielding for this reason is likely to be followed by another yield
		// very soon after so there is no point garthing this information
		if reason != mapper.YieldSyncWithVCS {
			// there's an assumption here that SortedLocals is sorted by variable name
			var prev string
			for _, varb := range src.SortedLocals.Variables {
				if prev == varb.Name {
					continue
				}
				if varb.find(ln) {
					locals = append(locals, varb.SourceVariable)
					prev = varb.Name
				}
			}
		}
	})

	dev.yieldStateLock.Lock()
	defer dev.yieldStateLock.Unlock()

	dev.yieldState.InstructionPC = instructionPC
	dev.yieldState.InstructionLine = ln
	dev.yieldState.Reason = reason

	// clear list of local variables from previous yield and extend with new
	// list of locals
	dev.yieldState.LocalVariables = dev.yieldState.LocalVariables[:0]
	dev.yieldState.LocalVariables = append(dev.yieldState.LocalVariables, locals...)
}

// BorrowYieldState will lock the illegal access log for the duration of the
// supplied fucntion, which will be executed with the illegal access log as an
// argument.
func (dev *Developer) BorrowYieldState(f func(*YieldState)) {
	dev.yieldStateLock.Lock()
	defer dev.yieldStateLock.Unlock()
	f(&dev.yieldState)
}
