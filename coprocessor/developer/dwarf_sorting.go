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

type sortMethods int

const (
	sortFunction sortMethods = iota
	sortLine
	sortLoad
	sortAverage
	sortMax
)

// SortedLines orders all the source lines in order of computational expense
type SortedLines struct {
	Lines      []*SourceLine
	method     sortMethods
	descending bool
}

func (e SortedLines) Sort() {
	sort.Stable(e)
}

func (e *SortedLines) SortByFrameCycles(descending bool) {
	e.descending = descending
	e.method = sortLoad
	sort.Stable(e)
}

func (e *SortedLines) SortByAverageCycles(descending bool) {
	e.descending = descending
	e.method = sortAverage
	sort.Stable(e)
}

func (e *SortedLines) SortByMaxCycles(descending bool) {
	e.descending = descending
	e.method = sortMax
	sort.Stable(e)
}

func (e *SortedLines) SortByLineAndFunction(descending bool) {
	e.descending = descending

	e.method = sortLine
	sort.Stable(e)

	e.method = sortFunction
	sort.Stable(e)
}

// Len implements sort.Interface.
func (e SortedLines) Len() int {
	return len(e.Lines)
}

// Less implements sort.Interface.
func (e SortedLines) Less(i int, j int) bool {
	switch e.method {
	case sortFunction:
		if e.descending {
			return e.Lines[i].Function.Name > e.Lines[j].Function.Name
		}
		return e.Lines[i].Function.Name < e.Lines[j].Function.Name
	case sortLine:
		if e.descending {
			return e.Lines[i].LineNumber > e.Lines[j].LineNumber
		}
		return e.Lines[i].LineNumber < e.Lines[j].LineNumber
	case sortLoad:
		if e.descending {
			return e.Lines[i].Stats.load > e.Lines[j].Stats.load
		}
		return e.Lines[i].Stats.load < e.Lines[j].Stats.load
	case sortAverage:
		if e.descending {
			return e.Lines[i].Stats.avgLoad > e.Lines[j].Stats.avgLoad
		}
		return e.Lines[i].Stats.avgLoad < e.Lines[j].Stats.avgLoad
	case sortMax:
		if e.descending {
			return e.Lines[i].Stats.maxLoad > e.Lines[j].Stats.maxLoad
		}
		return e.Lines[i].Stats.maxLoad < e.Lines[j].Stats.maxLoad
	}

	return false
}

// Swap implements sort.Interface.
func (e SortedLines) Swap(i int, j int) {
	e.Lines[i], e.Lines[j] = e.Lines[j], e.Lines[i]
}

// SortedFunctions orders all the source lines in order of computationally expense
type SortedFunctions struct {
	Functions  []*SourceFunction
	method     sortMethods
	descending bool
}

func (e SortedFunctions) Sort() {
	sort.Stable(e)
}

func (e *SortedFunctions) SortByFrameCycles(descending bool) {
	e.descending = descending
	e.method = sortLoad
	sort.Stable(e)
}

func (e *SortedFunctions) SortByAverageCycles(descending bool) {
	e.descending = descending
	e.method = sortAverage
	sort.Stable(e)
}

func (e *SortedFunctions) SortByMaxCycles(descending bool) {
	e.descending = descending
	e.method = sortMax
	sort.Stable(e)
}

func (e *SortedFunctions) SortByFunction(descending bool) {
	e.descending = descending
	e.method = sortFunction
	sort.Stable(e)
}

// Len implements sort.Interface.
func (e SortedFunctions) Len() int {
	return len(e.Functions)
}

// Less implements sort.Interface.
func (e SortedFunctions) Less(i int, j int) bool {
	switch e.method {
	case sortFunction:
		if e.descending {
			return e.Functions[i].Name > e.Functions[j].Name
		}
		return e.Functions[i].Name < e.Functions[j].Name
	case sortLoad:
		if e.descending {
			return e.Functions[i].Stats.load > e.Functions[j].Stats.load
		}
		return e.Functions[i].Stats.load < e.Functions[j].Stats.load
	case sortAverage:
		if e.descending {
			return e.Functions[i].Stats.avgLoad > e.Functions[j].Stats.avgLoad
		}
		return e.Functions[i].Stats.avgLoad < e.Functions[j].Stats.avgLoad
	case sortMax:
		if e.descending {
			return e.Functions[i].Stats.maxLoad > e.Functions[j].Stats.maxLoad
		}
		return e.Functions[i].Stats.maxLoad < e.Functions[j].Stats.maxLoad
	}

	return false
}

// Swap implements sort.Interface.
func (e SortedFunctions) Swap(i int, j int) {
	e.Functions[i], e.Functions[j] = e.Functions[j], e.Functions[i]
}
