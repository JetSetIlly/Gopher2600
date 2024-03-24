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
	"sort"

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/profiling"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
)

// Profiling implements the coprocessor.CartCoProcDeveloper interface.
func (dev *Developer) Profiling() *coprocessor.CartCoProcProfiler {
	if dev.source == nil {
		return nil
	}
	return &dev.profiler
}

// StartProfiling implements the coprocessor.CartCoProcDeveloper interface.
func (dev *Developer) StartProfiling() {
	if dev.source == nil {
		return
	}

	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	dev.yieldStateLock.Lock()
	defer dev.yieldStateLock.Unlock()

	if dev.yieldState.Reason != coprocessor.YieldProgramEnded {
		return
	}

	dev.callstackLock.Lock()
	defer dev.callstackLock.Unlock()

	dev.callstack.Stack = dev.callstack.Stack[:0]

	// first entry in the callstack is always the entry function
	dev.callstack.Stack = append(dev.callstack.Stack, dev.source.DriverSourceLine)
}

// ProcessProfiling implements the coprocessor.CartCoProcDeveloper interface.
func (dev *Developer) ProcessProfiling() {
	if dev.source == nil {
		return
	}

	// empty slice at end of function under all circumstances
	defer func() {
		dev.profiler.Entries = dev.profiler.Entries[:0]
	}()

	// accumulate function will be called with the correct KernelVCS
	accumulate := func(focus profiling.Focus) {
		dev.sourceLock.Lock()
		defer dev.sourceLock.Unlock()

		dev.callstackLock.Lock()
		defer dev.callstackLock.Unlock()

		for _, p := range dev.profiler.Entries {
			// line of executed instruction. every instruction should have an
			// associated line/function. if it does not then we assume it is in
			// the driver function
			ln, ok := dev.source.LinesByAddress[uint64(p.Addr)]
			if !ok {
				ln = dev.source.DriverSourceLine
				dev.source.LinesByAddress[uint64(p.Addr)] = ln
			}

			// update callstack for line only if the function for the line is
			// not a stub function and not the main function
			if !ln.Function.IsStub() && ln.Function != dev.source.MainFunction {
				l := len(dev.callstack.Stack) - 1
				prevCallStack := dev.callstack.Stack[l]

				// change callstack if function has changed
				if ln.Function != prevCallStack.Function {
					var popped bool

					// try to pop entry from callstack
					for i := 1; i < l && !popped; i++ {
						if ln.Function == dev.callstack.Stack[l-i].Function {
							chop := dev.callstack.Stack[l-i+1:]
							dev.callstack.Stack = dev.callstack.Stack[:l-i+1]

							// flag functions which look like they are part of an
							// optimised call stack
							if len(chop) > 1 {
								for _, ln := range chop {
									ln.Function.OptimisedCallStack = true
								}
							}

							// setting popped will cause the loop to end early
							popped = true
						}
					}

					// push function on to callstack if we haven't popped
					if !popped {
						dev.callstack.Stack = append(dev.callstack.Stack, ln)

						// create/update callers list for function
						var n int
						l, ok := dev.callstack.Callers[ln.Function.Name]
						if ok {
							n = sort.Search(len(l), func(i int) bool {
								return ln == l[i]
							})
						}

						if !ok || (n > len(l) && l[n] != dev.prevProfileLine) {
							l = append(l, dev.prevProfileLine)
							sort.Slice(l, func(i, j int) bool {
								return l[i].Function.Name < l[j].Function.Name
							})
							dev.callstack.Callers[ln.Function.Name] = l
						}

						// increase count of how many times this function has
						// been called this frame
						ln.Function.NumCalls.Call(focus)
					}
				}
			}

			// check that this function looks like it has been called at least once
			ln.Function.NumCalls.Check(focus)

			// accumulate counts for line (and the line's function)
			dev.source.ExecutionProfile(ln, p.Cycles, focus)

			// accumulate ancestor functions too
			for _, ln := range dev.callstack.Stack {
				dev.source.ExecutionProfileCumulative(ln.Function, p.Cycles, focus)
			}

			// record line for future comparison
			dev.prevProfileLine = ln
		}
	}

	// decide which profiling focus to use, based on the information we get from
	// the television interface
	//
	// note that our VBLANK signal check won't have any false positives if the
	// VBLANK is set during the visible part of the screen

	coords := dev.tv.GetCoords()
	frameInfo := dev.tv.GetFrameInfo()

	if frameInfo.Stable {
		if coords.Scanline == frameInfo.VisibleTop {
			if dev.tv.GetLastSignal()&signal.VBlank == signal.VBlank {
				accumulate(profiling.FocusVBLANK)
			} else {
				accumulate(profiling.FocusScreen)
			}
		} else if coords.Scanline == frameInfo.VisibleBottom {
			if dev.tv.GetLastSignal()&signal.VBlank == signal.VBlank {
				accumulate(profiling.FocusOverscan)
			} else {
				accumulate(profiling.FocusScreen)
			}
		} else if coords.Scanline < frameInfo.VisibleTop-1 {
			accumulate(profiling.FocusVBLANK)
		} else if coords.Scanline < frameInfo.VisibleBottom {
			accumulate(profiling.FocusScreen)
		} else {
			accumulate(profiling.FocusOverscan)
		}
	} else {
		accumulate(profiling.FocusAll)
	}
}
