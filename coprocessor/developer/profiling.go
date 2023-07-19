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

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

// Profiling implements the mapper.CartCoProcDeveloper interface.
func (dev *Developer) Profiling() *mapper.CartCoProcProfiler {
	if dev.source == nil {
		return nil
	}
	return &dev.profiler
}

// StartProfiling implements the mapper.CartCoProcDeveloper interface.
func (dev *Developer) StartProfiling() {
	if dev.disabledExpensive || dev.source == nil {
		return
	}

	dev.yieldStateLock.Lock()
	defer dev.yieldStateLock.Unlock()

	if dev.yieldState.Reason != mapper.YieldProgramEnded {
		return
	}

	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	dev.source.callStack.stack = dev.source.callStack.stack[:0]

	// first entry in the callstack is always the entry function
	dev.source.callStack.stack = append(dev.source.callStack.stack, dev.source.driverSourceLine)
}

// ProcessProfiling implements the mapper.CartCoProcDeveloper interface.
func (dev *Developer) ProcessProfiling() {
	if dev.disabledExpensive {
		return
	}

	// accumulate function will be called with the correct KernelVCS
	accumulate := func(k KernelVCS) {
		if dev.source == nil {
			return
		}

		dev.sourceLock.Lock()
		defer dev.sourceLock.Unlock()

		for _, p := range dev.profiler.Entries {
			l := len(dev.source.callStack.stack)
			lastLn := dev.source.callStack.stack[l-1]

			// line of executed instruction. every instruction should have an
			// associated line/function. if it does not then we assume it is in
			// the entry function
			ln, ok := dev.source.LinesByAddress[uint64(p.Addr)]
			if !ok {
				ln = dev.source.driverSourceLine
				dev.source.LinesByAddress[uint64(p.Addr)] = ln
			}

			// if function has changed
			if ln != lastLn {
				popped := false

				// try to pop
				var i int
				for i = 1; i <= l; i++ {
					if ln.Function == dev.source.callStack.stack[l-i].Function {
						chop := dev.source.callStack.stack[l-i+1:]
						dev.source.callStack.stack = dev.source.callStack.stack[:l-i+1]
						popped = true

						// flag functions which look like they are part of an
						// optimised call stack
						if len(chop) > 1 {
							for _, ln := range chop {
								ln.Function.OptimisedCallStack = true
							}
						}

						break // for loop
					}
				}

				// push function on to callstack if we haven't popped
				if !popped {
					dev.source.callStack.stack = append(dev.source.callStack.stack, ln)

					// there is always at least one entry in the functions callstack so we can confidently
					// subtract two from the length after the append above
					// prev := dev.source.CallStack.functions[len(dev.source.CallStack.functions)-2]

					// create/update callers list for function
					var n int
					l, ok := dev.source.callStack.callers[ln.Function.Name]
					if ok {
						n = sort.Search(len(l), func(i int) bool {
							return ln == l[i]
						})
					}
					if !ok || (n > len(l) && l[n] != dev.source.callStack.prevLine) {
						l = append(l, dev.source.callStack.prevLine)
						sort.Slice(l, func(i, j int) bool {
							return l[i].Function.Name < l[j].Function.Name
						})
						dev.source.callStack.callers[ln.Function.Name] = l
					}
				}

			}

			// accumulate counts for line (and the line's function)
			dev.source.executionProfile(ln, p.Cycles, k)

			// accumulate ancestor functions too
			for _, ln := range dev.source.callStack.stack {
				dev.source.executionProfileCumulative(ln.Function, p.Cycles, k)
			}

			// record line for future comparison
			dev.source.callStack.prevLine = ln
		}
	}

	// checking to see if kernel has changed
	if dev.frameInfo.Stable {
		c := dev.tv.GetCoords()

		if c.Scanline <= dev.frameInfo.VisibleTop-1 {
			accumulate(KernelVBLANK)
		} else if c.Scanline <= dev.frameInfo.VisibleBottom {
			accumulate(KernelScreen)
		} else {
			accumulate(KernelOverscan)
		}
	} else {
		accumulate(KernelUnstable)
	}

	// empty slice
	dev.profiler.Entries = dev.profiler.Entries[:0]
}

// KernelVCS indicates the 2600 kernel that is associated with a source function
// or source line.
type KernelVCS int

// List of KernelVCS values.
const (
	KernelAny      KernelVCS = 0x00
	KernelScreen   KernelVCS = 0x01
	KernelVBLANK   KernelVCS = 0x02
	KernelOverscan KernelVCS = 0x04

	// code that is run while the television is in an unstable state
	KernelUnstable KernelVCS = 0x08
)

func (k KernelVCS) String() string {
	switch k {
	case KernelScreen:
		return "Screen"
	case KernelVBLANK:
		return "VBLANK"
	case KernelOverscan:
		return "Overscan"
	case KernelUnstable:
		return "ROM Setup"
	}

	return "Any"
}

// List of KernelVCS values as strings
var AvailableInKernelOptions = []string{"Any", "VBLANK", "Screen", "Overscan", "ROM Setup"}

func (src *Source) executionProfile(ln *SourceLine, ct float32, kernel KernelVCS) {
	// indicate that execution profile has changed
	src.ExecutionProfileChanged = true

	fn := ln.Function

	ln.Stats.Overall.count += ct
	fn.FlatStats.Overall.count += ct
	src.Stats.Overall.count += ct

	ln.Kernel |= kernel
	fn.Kernel |= kernel
	if fn.DeclLine != nil {
		fn.DeclLine.Kernel |= kernel
	}

	switch kernel {
	case KernelVBLANK:
		ln.Stats.VBLANK.count += ct
		fn.FlatStats.VBLANK.count += ct
		src.Stats.VBLANK.count += ct
	case KernelScreen:
		ln.Stats.Screen.count += ct
		fn.FlatStats.Screen.count += ct
		src.Stats.Screen.count += ct
	case KernelOverscan:
		ln.Stats.Overscan.count += ct
		fn.FlatStats.Overscan.count += ct
		src.Stats.Overscan.count += ct
	case KernelUnstable:
		ln.Stats.ROMSetup.count += ct
		fn.FlatStats.ROMSetup.count += ct
		src.Stats.ROMSetup.count += ct
	}
}

func (src *Source) executionProfileCumulative(fn *SourceFunction, ct float32, kernel KernelVCS) {
	// indicate that execution profile has changed
	src.ExecutionProfileChanged = true

	fn.CumulativeStats.Overall.count += ct

	switch kernel {
	case KernelVBLANK:
		fn.CumulativeStats.VBLANK.count += ct
	case KernelScreen:
		fn.CumulativeStats.Screen.count += ct
	case KernelOverscan:
		fn.CumulativeStats.Overscan.count += ct
	case KernelUnstable:
		fn.CumulativeStats.ROMSetup.count += ct
	}
}
