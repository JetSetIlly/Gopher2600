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
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

// CallStack maintains information about function calls and the order in which
// they happen.
type CallStack struct {
	// call stack of running program
	functions []*SourceFunction
}

func (cs *CallStack) String() string {
	l := len(cs.functions)
	if l == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(cs.functions[0].Name)
	for i := 1; i < l; i++ {
		b.WriteString(" -> ")
		b.WriteString(cs.functions[i].Name)
	}
	return b.String()
}

// Profiling implements the mapper.CartCoProcDeveloper interface.
func (dev *Developer) Profiling() *mapper.CartCoProcProfiler {
	return &dev.profiler
}

// StartProfiling implements the mapper.CartCoProcDeveloper interface.
func (dev *Developer) StartProfiling() {
	if dev.disabled || dev.source == nil {
		return
	}

	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	dev.source.CallStack.functions = dev.source.CallStack.functions[:0]

	// first entry in the callstack is always the entry function
	dev.source.CallStack.functions = append(dev.source.CallStack.functions, dev.source.Functions[entryFunction])
}

// ProcessProfiling implements the mapper.CartCoProcDeveloper interface.
func (dev *Developer) ProcessProfiling() {
	if dev.disabled || dev.source == nil {
		return
	}

	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	dev.profileProcess()
}

// process cycle counts for (non-zero) addresses in dev.profiledAddresses
func (dev *Developer) profileProcess() {
	accumulate := func(k KernelVCS) {
		for _, p := range dev.profiler.Entries {
			l := len(dev.source.CallStack.functions)
			lastFn := dev.source.CallStack.functions[l-1]

			// line of executed instruction. every instruction should have an
			// associated line/function. if it does not then we assume it is in
			// the entry function
			ln, ok := dev.source.linesByAddress[uint64(p.Addr)]
			if !ok {
				ln = dev.source.entryLine
				dev.source.linesByAddress[uint64(p.Addr)] = ln
			}

			// if function has changed
			if ln.Function != lastFn {
				popped := false

				// try to pop
				for i := 1; i < l; i++ {
					if ln.Function == dev.source.CallStack.functions[l-i] {
						dev.source.CallStack.functions = dev.source.CallStack.functions[:l-i+1]
						popped = true
						break // for loop
					}
				}

				// push function on to callstack if we haven't popped
				if !popped {
					dev.source.CallStack.functions = append(dev.source.CallStack.functions, ln.Function)
				}
			}

			// accumulate counts for line (and the line's function)
			dev.source.executionProfile(ln, p.Cycles, k)

			// accumulate ancestor functions too
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

	// empty array
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

	ln.Stats.count += ct
	ln.Function.Stats.count += ct
	src.Stats.count += ct

	ln.Kernel |= kernel
	ln.Function.Kernel |= kernel
	if ln.Function.DeclLine != nil {
		ln.Function.DeclLine.Kernel |= kernel
	}

	switch kernel {
	case KernelVBLANK:
		ln.StatsVBLANK.count += ct
		ln.Function.StatsVBLANK.count += ct
		src.StatsVBLANK.count += ct
	case KernelScreen:
		ln.StatsScreen.count += ct
		ln.Function.StatsScreen.count += ct
		src.StatsScreen.count += ct
	case KernelOverscan:
		ln.StatsOverscan.count += ct
		ln.Function.StatsOverscan.count += ct
		src.StatsOverscan.count += ct
	case KernelUnstable:
		ln.StatsROMSetup.count += ct
		ln.Function.StatsROMSetup.count += ct
		src.StatsROMSetup.count += ct
	}
}
