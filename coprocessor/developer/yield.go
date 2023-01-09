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
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

// YieldedLocal supplements a SourceVariableLocal with additional information
// about the variable under current yield conditions.
type YieldedLocal struct {
	*SourceVariableLocal

	// whether this specific local variable is in resolvable range
	InRange bool
}

func (local *YieldedLocal) String() string {
	if local.ErrorOnResolve != nil {
		return fmt.Sprintf("%s = %s", local.Name, local.ErrorOnResolve)
	}
	if local.InRange {
		return local.SourceVariableLocal.String()
	}
	return fmt.Sprintf("%s = out of scope", local.Name)
}

// YieldState records the most recent yield.
type YieldState struct {
	InstructionPC  uint32
	Reason         mapper.YieldReason
	LocalVariables []*YieldedLocal
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
	var locals []*YieldedLocal

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

		// candidate is the variable that we want to add but are not sure if
		// there's a better candidate later in the sorted list
		var candidate *YieldedLocal
		commitCandidate := func() {
			if candidate != nil {
				locals = append(locals, candidate)
				candidate = nil
			}
		}

		// there's an assumption here that SortedLocals is sorted by variable name
		var id string
		var prevId string

		for _, local := range src.SortedLocals.Locals {
			// if the variable is in the current function then we always add it
			// to the list of local variables, even if it's not resolvable. in
			// those cases adding it to the list tells the user that the
			// debugger knows about the variable but that it can't be resolved.
			// dividing all local variables by function like this is better
			// than being strict about scoping rules - if a variable is out of
			// scope then it is not resolvable
			if ln.Function == local.DeclLine.Function {
				id = local.id()
				if prevId != id {
					commitCandidate()
				}
				prevId = id

				// we must use currentPC to test whether a local variable is in
				// range because although we're reporting that the instructionPC is
				// the breakpoint the machine is in the state defined by currentPC
				inRange := local.Range.InRange(uint64(currentPC))

				// add new YieldedLocal if a variable of this name has not been
				// added to list of locals already
				//
				// (assumes a list sorted by name, which it should be because
				// we are inserting from a sorted list)
				if len(locals) == 0 || id != locals[len(locals)-1].Name {
					candidate = &YieldedLocal{
						SourceVariableLocal: local,
						InRange:             inRange,
					}

					if inRange {
						commitCandidate()
					}
				}
			}
		}

		commitCandidate()

		// update all globals (locals are updated below)
		src.UpdateGlobalVariables()
	})

	dev.BorrowYieldState(func(yld *YieldState) {
		yld.InstructionPC = instructionPC
		yld.Reason = reason

		// clear list of local variables from previous yield
		yld.LocalVariables = yld.LocalVariables[:0]

		// filter new list of local variables for duplicates
		if len(locals) > 0 {
			yld.LocalVariables = append(yld.LocalVariables, locals[0])
			for _, local := range locals[1:] {
				prev := yld.LocalVariables[len(yld.LocalVariables)-1]
				if prev.Name == local.Name {
					// replace previous appended local variable with new local
					// variable if the new variable is resolving and the
					// previous varaible is not
					//
					// note that we're not checking for the case were two
					// locals with the same name are resolving because that
					// should not be possible with C like languages
					if !prev.InRange && local.InRange {
						yld.LocalVariables[len(yld.LocalVariables)-1] = local
					}
				} else {
					yld.LocalVariables = append(yld.LocalVariables, local)
				}
			}
		}

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
