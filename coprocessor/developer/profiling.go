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

// CallStack maintains information about function calls and the order in which
// they happen.
type CallStack struct {
	// call stack of running program
	functions []*SourceFunction

	// call stack ptr points to the last entry that was filled
	ptr int

	// whether the call stack has remained sane during profiling
	Unreliable bool
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

	if dev.source.CallStack.ptr > 1 {
		dev.source.CallStack.Unreliable = true
	}

	dev.source.CallStack.functions = dev.source.CallStack.functions[:0]
	dev.source.CallStack.functions = append(dev.source.CallStack.functions, dev.source.noSourceLine.Function)
	dev.source.CallStack.ptr = 0
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
			// line of executed instruction.every instruction should have an
			// associated line/function. if it does not then we use the
			// noSourceLine instance as a placeholder
			ln, ok := dev.source.linesByAddress[uint64(p.Addr)]
			if !ok {
				ln = dev.source.noSourceLine
				dev.source.linesByAddress[uint64(p.Addr)] = ln
			}

			// the underlying function for the line
			fn := ln.Function

			// if function has changed then either add an entry or remove an
			// entry depending on whether the function has been seen recently.
			if fn != dev.source.CallStack.functions[dev.source.CallStack.ptr] {
				if dev.source.CallStack.ptr > 0 && fn != dev.source.CallStack.functions[dev.source.CallStack.ptr-1] {
					dev.source.CallStack.ptr--
				} else {
					dev.source.CallStack.functions = append(dev.source.CallStack.functions, fn)
					dev.source.CallStack.ptr++
				}
			}

			// accumulate for the current function
			dev.source.executionProfile(ln, p.Cycles, k)

			// accumulate ancestor functions too
			for i := dev.source.CallStack.ptr - 1; i >= 0; i-- {
			}
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
	ln.Function.DeclLine.Kernel |= kernel

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
