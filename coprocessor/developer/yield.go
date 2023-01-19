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
	InstructionPC  uint32
	Reason         mapper.YieldReason
	LocalVariables []*SourceVariableLocal
}

// Cmp returns true if two YieldStates are equal.
func (y *YieldState) Cmp(w *YieldState) bool {
	return y.InstructionPC == w.InstructionPC && y.Reason == w.Reason
}

// OnYield implements the mapper.CartCoProcDeveloper interface.
func (dev *Developer) OnYield(instructionPC uint32, currentPC uint32, reason mapper.YieldReason) {
	// do nothing if yield reason is YieldSyncWithVCS
	//
	// yielding for this reason is likely to be followed by another yield
	// very soon after so there is no point gathering this information
	if reason == mapper.YieldSyncWithVCS {
		dev.BorrowYieldState(func(yld *YieldState) {
			yld.InstructionPC = instructionPC
			yld.Reason = reason
			yld.LocalVariables = yld.LocalVariables[:0]
		})

		return
	}

	var ln *SourceLine
	var locals []*SourceVariableLocal

	// using BorrowSource (this is better than just acquiring the lock because we want to make sure
	// the lock is released if there is an error and the code panics)
	dev.BorrowSource(func(src *Source) {
		// make sure that src is valid
		if src == nil {
			return
		}

		ln = src.FindSourceLine(instructionPC)
		if ln == nil {
			return
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

		// the chosen local variable
		var chosenLocal *SourceVariableLocal

		// choose function that covers the smallest (most specific) range in which startAddr
		// appears
		chosenSize := ^uint64(0)

		// function to add chosen local variable to the yield
		commitChosen := func() {
			locals = append(locals, chosenLocal)
			chosenLocal = nil
			chosenSize = ^uint64(0)
		}

		// there's an assumption here that SortedLocals is sorted by variable name
		for _, local := range src.SortedLocals.Locals {
			// append chosen local variable
			if chosenLocal != nil && chosenLocal.Name != local.Name {
				commitChosen()
			}

			// ignore variables that are not declared to be in the same
			// function as the break line. this can happen for inlined
			// functions when function ranges overlap
			if local.DeclLine.Function == ln.Function {
				// we must use currentPC to test whether a local variable is in
				// range because, although we're reporting that the instructionPC is
				// the breakpoint, the machine is in the state defined by currentPC
				if local.Range.InRange(uint64(currentPC)) {
					if local.Range.Size() < chosenSize {
						chosenLocal = local
						chosenSize = local.Range.Size()
					}
				}
			}
		}

		// append chosen local variable
		if chosenLocal != nil {
			commitChosen()
		}

		// update all globals (locals are updated below)
		src.UpdateGlobalVariables()
	})

	dev.BorrowYieldState(func(yld *YieldState) {
		yld.InstructionPC = instructionPC
		yld.Reason = reason

		// clear list of local variables from previous yield
		yld.LocalVariables = yld.LocalVariables[:0]
		yld.LocalVariables = append(yld.LocalVariables, locals...)

		// update all locals (globals are updated above)
		for _, local := range yld.LocalVariables {
			local.Update()
		}
	})
}

// BorrowYieldState will lock the illegal access log for the duration of the
// supplied fucntion, which will be executed with the illegal access log as an
// argument.
func (dev *Developer) BorrowYieldState(f func(*YieldState)) {
	dev.yieldStateLock.Lock()
	defer dev.yieldStateLock.Unlock()
	f(&dev.yieldState)
}
