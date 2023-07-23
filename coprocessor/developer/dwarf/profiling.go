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

import "github.com/jetsetilly/gopher2600/coprocessor/developer/profiling"

func (src *Source) NewFrame() {
	// calling newFrame() on stats in a specific order. first the program, then
	// the functions and then the source lines.

	src.Stats.Overall.NewFrame(nil, nil)
	src.Stats.VBLANK.NewFrame(nil, nil)
	src.Stats.Screen.NewFrame(nil, nil)
	src.Stats.Overscan.NewFrame(nil, nil)

	for _, fn := range src.Functions {
		fn.FlatStats.Overall.NewFrame(&src.Stats.Overall, nil)
		fn.FlatStats.VBLANK.NewFrame(&src.Stats.VBLANK, nil)
		fn.FlatStats.Screen.NewFrame(&src.Stats.Screen, nil)
		fn.FlatStats.Overscan.NewFrame(&src.Stats.Overscan, nil)

		fn.CumulativeStats.Overall.NewFrame(&src.Stats.Overall, nil)
		fn.CumulativeStats.VBLANK.NewFrame(&src.Stats.VBLANK, nil)
		fn.CumulativeStats.Screen.NewFrame(&src.Stats.Screen, nil)
		fn.CumulativeStats.Overscan.NewFrame(&src.Stats.Overscan, nil)
	}

	// traverse the SortedLines list and update the FrameCyles values
	//
	// we prefer this over traversing the Lines list because we may hit a
	// SourceLine more than once. SortedLines contains unique entries.
	for _, ln := range src.SortedLines.Lines {
		ln.Stats.Overall.NewFrame(&src.Stats.Overall, &ln.Function.FlatStats.Overall)
		ln.Stats.VBLANK.NewFrame(&src.Stats.VBLANK, &ln.Function.FlatStats.VBLANK)
		ln.Stats.Screen.NewFrame(&src.Stats.Screen, &ln.Function.FlatStats.Screen)
		ln.Stats.Overscan.NewFrame(&src.Stats.Overscan, &ln.Function.FlatStats.Overscan)
	}
}

func (src *Source) ExecutionProfile(ln *SourceLine, ct float32, focus profiling.Focus) {
	// indicate that execution profile has changed
	src.ExecutionProfileChanged = true

	fn := ln.Function

	ln.Stats.Overall.Count += ct
	fn.FlatStats.Overall.Count += ct
	src.Stats.Overall.Count += ct

	ln.Kernel |= focus
	fn.Kernel |= focus
	if fn.DeclLine != nil {
		fn.DeclLine.Kernel |= focus
	}

	switch focus {
	case profiling.FocusAll:
		// deliberately ignore
	case profiling.FocusVBLANK:
		ln.Stats.VBLANK.Count += ct
		fn.FlatStats.VBLANK.Count += ct
		src.Stats.VBLANK.Count += ct
	case profiling.FocusScreen:
		ln.Stats.Screen.Count += ct
		fn.FlatStats.Screen.Count += ct
		src.Stats.Screen.Count += ct
	case profiling.FocusOverscan:
		ln.Stats.Overscan.Count += ct
		fn.FlatStats.Overscan.Count += ct
		src.Stats.Overscan.Count += ct
	default:
		panic("unknown focus type")
	}
}

func (src *Source) ExecutionProfileCumulative(fn *SourceFunction, ct float32, focus profiling.Focus) {
	// indicate that execution profile has changed
	src.ExecutionProfileChanged = true

	fn.CumulativeStats.Overall.Count += ct

	switch focus {
	case profiling.FocusAll:
		// deliberately ignore
	case profiling.FocusVBLANK:
		fn.CumulativeStats.VBLANK.Count += ct
	case profiling.FocusScreen:
		fn.CumulativeStats.Screen.Count += ct
	case profiling.FocusOverscan:
		fn.CumulativeStats.Overscan.Count += ct
	default:
		panic("unknown focus type")
	}
}

// ResetStatistics resets all performance statistics.
func (src *Source) ResetStatistics() {
	for i := range src.Functions {
		src.Functions[i].Kernel = profiling.FocusAll
		src.Functions[i].FlatStats.Overall.Reset()
		src.Functions[i].FlatStats.VBLANK.Reset()
		src.Functions[i].FlatStats.Screen.Reset()
		src.Functions[i].FlatStats.Overscan.Reset()
		src.Functions[i].CumulativeStats.Overall.Reset()
		src.Functions[i].CumulativeStats.VBLANK.Reset()
		src.Functions[i].CumulativeStats.Screen.Reset()
		src.Functions[i].CumulativeStats.Overscan.Reset()
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
