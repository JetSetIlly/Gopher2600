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
)

// SortedLines orders all the source lines in order of computational expense
type SortedLines struct {
	Lines []*SourceLine

	// 0 = frame cycles (default)
	// 1 = average cycles
	// 2 = function name
	// 3 = line number
	method int

	// direction of the sort
	descending bool
}

func (e SortedLines) Sort() {
	sort.Stable(e)
}

func (e *SortedLines) SortByFrameCycles(descending bool) {
	e.descending = descending
	e.method = 0
	sort.Stable(e)
}

func (e *SortedLines) SortByAverageCycles(descending bool) {
	e.descending = descending
	e.method = 1
	sort.Stable(e)
}

func (e *SortedLines) SortByLineAndFunction(descending bool) {
	e.descending = descending

	e.method = 3
	sort.Stable(e)

	e.method = 2
	sort.Stable(e)
}

// Len implements sort.Interface.
func (e SortedLines) Len() int {
	return len(e.Lines)
}

// Less implements sort.Interface.
func (e SortedLines) Less(i int, j int) bool {
	switch e.method {
	case 1:
		if e.descending {
			return e.Lines[i].Stats.AvgCycles > e.Lines[j].Stats.AvgCycles
		}
		return e.Lines[i].Stats.AvgCycles < e.Lines[j].Stats.AvgCycles
	case 2:
		if e.descending {
			return e.Lines[i].Function.Name > e.Lines[j].Function.Name
		}
		return e.Lines[i].Function.Name < e.Lines[j].Function.Name
	case 3:
		if e.descending {
			return e.Lines[i].LineNumber > e.Lines[j].LineNumber
		}
		return e.Lines[i].LineNumber < e.Lines[j].LineNumber
	default:
		if e.descending {
			return e.Lines[i].Stats.FrameCycles > e.Lines[j].Stats.FrameCycles
		}
		return e.Lines[i].Stats.FrameCycles < e.Lines[j].Stats.FrameCycles
	}
}

// Swap implements sort.Interface.
func (e SortedLines) Swap(i int, j int) {
	e.Lines[i], e.Lines[j] = e.Lines[j], e.Lines[i]
}

// SortedFunctions orders all the source lines in order of computationally expense
type SortedFunctions struct {
	Functions []*SourceFunction

	// 0 = frame cycles (default)
	// 1 = average cycles
	// 2 = function name
	method int

	// whether the sort should be reversed or not
	descending bool
}

func (e SortedFunctions) Sort() {
	sort.Stable(e)
}

func (e *SortedFunctions) SortByFrameCycles(descending bool) {
	e.descending = descending
	e.method = 0
	sort.Stable(e)
}

func (e *SortedFunctions) SortByAverageCycles(descending bool) {
	e.descending = descending
	e.method = 1
	sort.Stable(e)
}

func (e *SortedFunctions) SortByFunction(descending bool) {
	e.descending = descending
	e.method = 2
	sort.Stable(e)
}

// Len implements sort.Interface.
func (e SortedFunctions) Len() int {
	return len(e.Functions)
}

// Less implements sort.Interface.
func (e SortedFunctions) Less(i int, j int) bool {
	switch e.method {
	case 1:
		if e.descending {
			return e.Functions[i].Stats.AvgCycles > e.Functions[j].Stats.AvgCycles
		}
		return e.Functions[i].Stats.AvgCycles < e.Functions[j].Stats.AvgCycles
	case 2:
		if e.descending {
			return e.Functions[i].Name > e.Functions[j].Name
		}
		return e.Functions[i].Name < e.Functions[j].Name
	default:
		if e.descending {
			return e.Functions[i].Stats.FrameCycles > e.Functions[j].Stats.FrameCycles
		}
		return e.Functions[i].Stats.FrameCycles < e.Functions[j].Stats.FrameCycles
	}
}

// Swap implements sort.Interface.
func (e SortedFunctions) Swap(i int, j int) {
	e.Functions[i], e.Functions[j] = e.Functions[j], e.Functions[i]
}
