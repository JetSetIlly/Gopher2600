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

package dwarf

import (
	"github.com/jetsetilly/gopher2600/coprocessor/developer/profiling"
)

func (src *Source) NewFrame(rewinding bool) {
	// calling newFrame() on stats in a specific order. first the program, then
	// the functions and then the source lines.

	src.Cycles.NewFrame(nil, nil, rewinding)

	for _, fn := range src.Functions {
		fn.FlatCycles.NewFrame(&src.Cycles, nil, rewinding)
		fn.CumulativeCycles.NewFrame(&src.Cycles, nil, rewinding)
		fn.NumCalls.NewFrame(rewinding)
		fn.CyclesPerCall.NewFrame(rewinding)
	}

	// traverse the SortedLines list and update the FrameCyles values
	//
	// we prefer this over traversing the Lines list because we may hit a
	// SourceLine more than once. SortedLines contains unique entries.
	for _, ln := range src.SortedLines.Lines {
		ln.Stats.NewFrame(&src.Cycles, &ln.Function.FlatCycles, rewinding)
	}
}

func (src *Source) ExecutionProfile(ln *SourceLine, ct float32, focus profiling.Focus) {
	// indicate that execution profile has changed
	src.StatsDirty = true

	fn := ln.Function

	ln.Stats.Overall.Cycle(ct)
	fn.FlatCycles.Overall.Cycle(ct)
	src.Cycles.Overall.Cycle(ct)

	ln.Kernel |= focus
	fn.Kernel |= focus
	if fn.DeclLine != nil {
		fn.DeclLine.Kernel |= focus
	}

	switch focus {
	case profiling.FocusAll:
		// deliberately ignore
	case profiling.FocusVBLANK:
		ln.Stats.VBLANK.Cycle(ct)
		fn.FlatCycles.VBLANK.Cycle(ct)
		src.Cycles.VBLANK.Cycle(ct)
	case profiling.FocusScreen:
		ln.Stats.Screen.Cycle(ct)
		fn.FlatCycles.Screen.Cycle(ct)
		src.Cycles.Screen.Cycle(ct)
	case profiling.FocusOverscan:
		ln.Stats.Overscan.Cycle(ct)
		fn.FlatCycles.Overscan.Cycle(ct)
		src.Cycles.Overscan.Cycle(ct)
	default:
		panic("unknown focus type")
	}
}

func (src *Source) ExecutionProfileCumulative(fn *SourceFunction, ct float32, focus profiling.Focus) {
	// indicate that execution profile has changed
	src.StatsDirty = true

	fn.CumulativeCycles.Overall.Cycle(ct)

	switch focus {
	case profiling.FocusAll:
		// deliberately ignore
	case profiling.FocusVBLANK:
		fn.CumulativeCycles.VBLANK.Cycle(ct)
	case profiling.FocusScreen:
		fn.CumulativeCycles.Screen.Cycle(ct)
	case profiling.FocusOverscan:
		fn.CumulativeCycles.Overscan.Cycle(ct)
	default:
		panic("unknown focus type")
	}
}

// ResetStatistics resets all performance statistics.
func (src *Source) ResetStatistics() {
	for i := range src.Functions {
		src.Functions[i].Kernel = profiling.FocusAll
		src.Functions[i].FlatCycles.Reset()
		src.Functions[i].CumulativeCycles.Reset()
		src.Functions[i].NumCalls.Reset()
		src.Functions[i].CyclesPerCall.Reset()
		src.Functions[i].OptimisedCallStack = false
	}
	for i := range src.LinesByAddress {
		src.LinesByAddress[i].Kernel = profiling.FocusAll
		src.LinesByAddress[i].Stats.Reset()
	}
	src.Cycles.Reset()
}
