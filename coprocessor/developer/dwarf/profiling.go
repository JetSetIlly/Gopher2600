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

// NewFrame commits the accumulated profiling for the frame. The rewinding flag
// indicates that the emulation is in the rewinding state and that some data
// should not be updated
func (src *Source) NewFrame(rewinding bool) {
	// calling newFrame() profiling data in a specific order
	src.Cycles.NewFrame(nil, nil, rewinding)

	var totalCyclesPerCall float32

	for _, fn := range src.Functions {
		fn.Cycles.NewFrame(&src.Cycles, nil, rewinding)
		fn.CumulativeCycles.NewFrame(&src.Cycles, nil, rewinding)
		fn.NumCalls.NewFrame(rewinding)
		fn.CyclesPerCall.NewFrame(rewinding)
		totalCyclesPerCall += fn.CyclesPerCall.Overall.FrameCount
	}

	for _, fn := range src.Functions {
		fn.CyclesPerCall.PostNewFrame(totalCyclesPerCall, rewinding)
	}

	// traverse the SortedLines list and update the FrameCyles values
	//
	// we prefer this over traversing the Lines list because we may hit a
	// SourceLine more than once. SortedLines contains unique entries.
	for _, ln := range src.SortedLines.Lines {
		ln.Cycles.NewFrame(&src.Cycles, &ln.Function.Cycles, rewinding)
	}
}

// ResetProfiling resets all profiling information
func (src *Source) ResetProfiling() {
	for i := range src.Functions {
		src.Functions[i].Kernel = profiling.FocusAll
		src.Functions[i].Cycles.Reset()
		src.Functions[i].CumulativeCycles.Reset()
		src.Functions[i].NumCalls.Reset()
		src.Functions[i].CyclesPerCall.Reset()
		src.Functions[i].OptimisedCallStack = false
	}
	for i := range src.LinesByAddress {
		src.LinesByAddress[i].Kernel = profiling.FocusAll
		src.LinesByAddress[i].Cycles.Reset()
	}
	src.Cycles.Reset()
}
