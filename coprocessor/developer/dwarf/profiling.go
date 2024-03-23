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

	src.Stats.Overall.NewFrame(nil, nil, rewinding)
	src.Stats.VBLANK.NewFrame(nil, nil, rewinding)
	src.Stats.Screen.NewFrame(nil, nil, rewinding)
	src.Stats.Overscan.NewFrame(nil, nil, rewinding)

	for _, fn := range src.Functions {
		fn.FlatCycles.Overall.NewFrame(&src.Stats.Overall, nil, rewinding)
		fn.FlatCycles.VBLANK.NewFrame(&src.Stats.VBLANK, nil, rewinding)
		fn.FlatCycles.Screen.NewFrame(&src.Stats.Screen, nil, rewinding)
		fn.FlatCycles.Overscan.NewFrame(&src.Stats.Overscan, nil, rewinding)

		fn.CumulativeCycles.Overall.NewFrame(&src.Stats.Overall, nil, rewinding)
		fn.CumulativeCycles.VBLANK.NewFrame(&src.Stats.VBLANK, nil, rewinding)
		fn.CumulativeCycles.Screen.NewFrame(&src.Stats.Screen, nil, rewinding)
		fn.CumulativeCycles.Overscan.NewFrame(&src.Stats.Overscan, nil, rewinding)

		fn.NumCalls.NewFrame(rewinding)
	}

	// traverse the SortedLines list and update the FrameCyles values
	//
	// we prefer this over traversing the Lines list because we may hit a
	// SourceLine more than once. SortedLines contains unique entries.
	for _, ln := range src.SortedLines.Lines {
		ln.Stats.Overall.NewFrame(&src.Stats.Overall, &ln.Function.FlatCycles.Overall, rewinding)
		ln.Stats.VBLANK.NewFrame(&src.Stats.VBLANK, &ln.Function.FlatCycles.VBLANK, rewinding)
		ln.Stats.Screen.NewFrame(&src.Stats.Screen, &ln.Function.FlatCycles.Screen, rewinding)
		ln.Stats.Overscan.NewFrame(&src.Stats.Overscan, &ln.Function.FlatCycles.Overscan, rewinding)
	}
}

func (src *Source) ExecutionProfile(ln *SourceLine, ct float32, focus profiling.Focus) {
	// indicate that execution profile has changed
	src.ExecutionProfileChanged = true

	fn := ln.Function

	ln.Stats.Overall.CycleCount += ct
	fn.FlatCycles.Overall.CycleCount += ct
	src.Stats.Overall.CycleCount += ct

	ln.Kernel |= focus
	fn.Kernel |= focus
	if fn.DeclLine != nil {
		fn.DeclLine.Kernel |= focus
	}

	switch focus {
	case profiling.FocusAll:
		// deliberately ignore
	case profiling.FocusVBLANK:
		ln.Stats.VBLANK.CycleCount += ct
		fn.FlatCycles.VBLANK.CycleCount += ct
		src.Stats.VBLANK.CycleCount += ct
	case profiling.FocusScreen:
		ln.Stats.Screen.CycleCount += ct
		fn.FlatCycles.Screen.CycleCount += ct
		src.Stats.Screen.CycleCount += ct
	case profiling.FocusOverscan:
		ln.Stats.Overscan.CycleCount += ct
		fn.FlatCycles.Overscan.CycleCount += ct
		src.Stats.Overscan.CycleCount += ct
	default:
		panic("unknown focus type")
	}
}

func (src *Source) ExecutionProfileCumulative(fn *SourceFunction, ct float32, focus profiling.Focus) {
	// indicate that execution profile has changed
	src.ExecutionProfileChanged = true

	fn.CumulativeCycles.Overall.CycleCount += ct

	switch focus {
	case profiling.FocusAll:
		// deliberately ignore
	case profiling.FocusVBLANK:
		fn.CumulativeCycles.VBLANK.CycleCount += ct
	case profiling.FocusScreen:
		fn.CumulativeCycles.Screen.CycleCount += ct
	case profiling.FocusOverscan:
		fn.CumulativeCycles.Overscan.CycleCount += ct
	default:
		panic("unknown focus type")
	}
}

// ResetStatistics resets all performance statistics.
func (src *Source) ResetStatistics() {
	for i := range src.Functions {
		src.Functions[i].Kernel = profiling.FocusAll
		src.Functions[i].FlatCycles.Overall.Reset()
		src.Functions[i].FlatCycles.VBLANK.Reset()
		src.Functions[i].FlatCycles.Screen.Reset()
		src.Functions[i].FlatCycles.Overscan.Reset()
		src.Functions[i].CumulativeCycles.Overall.Reset()
		src.Functions[i].CumulativeCycles.VBLANK.Reset()
		src.Functions[i].CumulativeCycles.Screen.Reset()
		src.Functions[i].CumulativeCycles.Overscan.Reset()
		src.Functions[i].NumCalls.Reset()
		src.Functions[i].OptimisedCallStack = false
	}
	for i := range src.LinesByAddress {
		src.LinesByAddress[i].Kernel = profiling.FocusAll
		src.LinesByAddress[i].Stats.Overall.Reset()
		src.LinesByAddress[i].Stats.VBLANK.Reset()
		src.LinesByAddress[i].Stats.Screen.Reset()
		src.LinesByAddress[i].Stats.Overscan.Reset()
	}
	src.Stats.Overall.Reset()
	src.Stats.VBLANK.Reset()
	src.Stats.Screen.Reset()
	src.Stats.Overscan.Reset()
}
