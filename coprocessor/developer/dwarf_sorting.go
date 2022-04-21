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
	"strings"
)

type sortMethods int

const (
	sortFunction sortMethods = iota
	sortFile
	sortLine
	sortFrameCyclesOverSource
	sortAverageCyclesOverSource
	sortMaxCyclesOverSource
	sortFrameCyclesOverFunction
	sortAverageCyclesOverFunction
	sortMaxCyclesOverFunction
)

type SortedLines struct {
	Lines      []*SourceLine
	method     sortMethods
	descending bool
}

func (e SortedLines) Sort() {
	sort.Stable(e)
}

func (e *SortedLines) SortByFile(descending bool) {
	e.descending = descending
	e.method = sortFile
	sort.Stable(e)
}

func (e *SortedLines) SortByLineNumber(descending bool) {
	e.descending = descending
	e.method = sortLine
	sort.Stable(e)
}

func (e *SortedLines) SortByFunction(descending bool) {
	e.descending = descending
	e.method = sortFunction
	sort.Stable(e)
}

func (e *SortedLines) SortByFrameLoadOverSource(descending bool) {
	e.descending = descending
	e.method = sortFrameCyclesOverSource
	sort.Stable(e)
}

func (e *SortedLines) SortByAverageLoadOverSource(descending bool) {
	e.descending = descending
	e.method = sortAverageCyclesOverSource
	sort.Stable(e)
}

func (e *SortedLines) SortByMaxLoadOverSource(descending bool) {
	e.descending = descending
	e.method = sortMaxCyclesOverSource
	sort.Stable(e)
}

func (e *SortedLines) SortByFrameLoadOverFunction(descending bool) {
	e.descending = descending
	e.method = sortFrameCyclesOverFunction
	sort.Stable(e)
}

func (e *SortedLines) SortByAverageLoadOverFunction(descending bool) {
	e.descending = descending
	e.method = sortAverageCyclesOverFunction
	sort.Stable(e)
}

func (e *SortedLines) SortByMaxLoadOverFunction(descending bool) {
	e.descending = descending
	e.method = sortMaxCyclesOverFunction
	sort.Stable(e)
}

func (e *SortedLines) SortByLineAndFunction(descending bool) {
	e.descending = descending
	e.method = sortLine
	sort.Stable(e)
	e.method = sortFunction
	sort.Stable(e)
}

func (e SortedLines) Len() int {
	return len(e.Lines)
}

func (e SortedLines) Less(i int, j int) bool {
	switch e.method {
	case sortFunction:
		if e.descending {
			return strings.ToUpper(e.Lines[i].Function.Name) > strings.ToUpper(e.Lines[j].Function.Name)
		}
		return strings.ToUpper(e.Lines[i].Function.Name) < strings.ToUpper(e.Lines[j].Function.Name)
	case sortFile:
		if e.descending {
			return e.Lines[i].Function.DeclLine.File.Filename > e.Lines[j].Function.DeclLine.File.Filename
		}
		return e.Lines[i].Function.DeclLine.File.Filename < e.Lines[j].Function.DeclLine.File.Filename
	case sortLine:
		if e.descending {
			return e.Lines[i].LineNumber > e.Lines[j].LineNumber
		}
		return e.Lines[i].LineNumber < e.Lines[j].LineNumber
	case sortFrameCyclesOverSource:
		if e.descending {
			return e.Lines[i].Stats.OverSource.Frame > e.Lines[j].Stats.OverSource.Frame
		}
		return e.Lines[i].Stats.OverSource.Frame > e.Lines[j].Stats.OverSource.Frame
	case sortAverageCyclesOverSource:
		if e.descending {
			return e.Lines[i].Stats.OverSource.Average > e.Lines[j].Stats.OverSource.Average
		}
		return e.Lines[i].Stats.OverSource.Average < e.Lines[j].Stats.OverSource.Average
	case sortMaxCyclesOverSource:
		if e.descending {
			return e.Lines[i].Stats.OverSource.Max > e.Lines[j].Stats.OverSource.Max
		}
		return e.Lines[i].Stats.OverSource.Max < e.Lines[j].Stats.OverSource.Max
	case sortFrameCyclesOverFunction:
		if e.descending {
			return e.Lines[i].Stats.OverFunction.Frame > e.Lines[j].Stats.OverFunction.Frame
		}
		return e.Lines[i].Stats.OverFunction.Frame > e.Lines[j].Stats.OverFunction.Frame
	case sortAverageCyclesOverFunction:
		if e.descending {
			return e.Lines[i].Stats.OverFunction.Average > e.Lines[j].Stats.OverFunction.Average
		}
		return e.Lines[i].Stats.OverFunction.Average < e.Lines[j].Stats.OverFunction.Average
	case sortMaxCyclesOverFunction:
		if e.descending {
			return e.Lines[i].Stats.OverFunction.Max > e.Lines[j].Stats.OverFunction.Max
		}
		return e.Lines[i].Stats.OverFunction.Max < e.Lines[j].Stats.OverFunction.Max
	}

	return false
}

func (e SortedLines) Swap(i int, j int) {
	e.Lines[i], e.Lines[j] = e.Lines[j], e.Lines[i]
}

type SortedFunctions struct {
	Functions  []*SourceFunction
	method     sortMethods
	descending bool

	functionComparison bool
}

func (e SortedFunctions) Sort() {
	sort.Stable(e)
}

func (e *SortedFunctions) SortByFile(descending bool) {
	e.descending = descending
	e.method = sortFile
	sort.Stable(e)
}

func (e *SortedFunctions) SortByFunction(descending bool) {
	e.descending = descending
	e.method = sortFunction
	sort.Stable(e)
}

func (e *SortedFunctions) SortByFrameCycles(descending bool) {
	e.descending = descending
	e.method = sortFrameCyclesOverSource
	sort.Stable(e)
}

func (e *SortedFunctions) SortByAverageCycles(descending bool) {
	e.descending = descending
	e.method = sortAverageCyclesOverSource
	sort.Stable(e)
}

func (e *SortedFunctions) SortByMaxCycles(descending bool) {
	e.descending = descending
	e.method = sortMaxCyclesOverSource
	sort.Stable(e)
}

func (e SortedFunctions) Len() int {
	return len(e.Functions)
}

func (e SortedFunctions) Less(i int, j int) bool {
	switch e.method {
	case sortFile:
		if e.descending {
			return e.Functions[i].DeclLine.File.Filename > e.Functions[j].DeclLine.File.Filename
		}
		return e.Functions[i].DeclLine.File.Filename < e.Functions[j].DeclLine.File.Filename
	case sortFunction:
		if e.descending {
			return strings.ToUpper(e.Functions[i].Name) > strings.ToUpper(e.Functions[j].Name)
		}
		return strings.ToUpper(e.Functions[i].Name) < strings.ToUpper(e.Functions[j].Name)
	case sortFrameCyclesOverSource:
		if e.descending {
			return e.Functions[i].Stats.OverSource.Frame > e.Functions[j].Stats.OverSource.Frame
		}
		return e.Functions[i].Stats.OverSource.Frame < e.Functions[j].Stats.OverSource.Frame
	case sortAverageCyclesOverSource:
		if e.descending {
			return e.Functions[i].Stats.OverSource.Average > e.Functions[j].Stats.OverSource.Average
		}
		return e.Functions[i].Stats.OverSource.Average < e.Functions[j].Stats.OverSource.Average
	case sortMaxCyclesOverSource:
		if e.descending {
			return e.Functions[i].Stats.OverSource.Max > e.Functions[j].Stats.OverSource.Max
		}
		return e.Functions[i].Stats.OverSource.Max < e.Functions[j].Stats.OverSource.Max
	}

	return false
}

func (e SortedFunctions) Swap(i int, j int) {
	e.Functions[i], e.Functions[j] = e.Functions[j], e.Functions[i]
}

type sortedVariableNames []string

func (v sortedVariableNames) Len() int {
	return len(v)
}

func (v sortedVariableNames) Less(i int, j int) bool {
	return strings.ToUpper(v[i]) < strings.ToUpper(v[j])
}

func (v sortedVariableNames) Swap(i int, j int) {
	v[i], v[j] = v[j], v[i]
}
