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
	kernel     InKernel
}

func (e SortedLines) Sort() {
	sort.Stable(e)
}

func (e *SortedLines) SetKernel(kernel InKernel) {
	e.kernel = kernel
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
	var iStats Stats
	var jStats Stats

	switch e.kernel {
	case InVBLANK:
		iStats = e.Lines[i].StatsVBLANK
		jStats = e.Lines[j].StatsVBLANK
	case InScreen:
		iStats = e.Lines[i].StatsScreen
		jStats = e.Lines[j].StatsScreen
	case InOverscan:
		iStats = e.Lines[i].StatsOverscan
		jStats = e.Lines[j].StatsOverscan
	default:
		iStats = e.Lines[i].Stats
		jStats = e.Lines[j].Stats
	}

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
			return iStats.OverSource.Frame > jStats.OverSource.Frame
		}
		return iStats.OverSource.Frame > jStats.OverSource.Frame
	case sortAverageCyclesOverSource:
		if e.descending {
			return iStats.OverSource.Average > jStats.OverSource.Average
		}
		return iStats.OverSource.Average < jStats.OverSource.Average
	case sortMaxCyclesOverSource:
		if e.descending {
			return iStats.OverSource.Max > jStats.OverSource.Max
		}
		return iStats.OverSource.Max < jStats.OverSource.Max
	case sortFrameCyclesOverFunction:
		if e.descending {
			return iStats.OverFunction.Frame > jStats.OverFunction.Frame
		}
		return iStats.OverFunction.Frame > jStats.OverFunction.Frame
	case sortAverageCyclesOverFunction:
		if e.descending {
			return iStats.OverFunction.Average > jStats.OverFunction.Average
		}
		return iStats.OverFunction.Average < jStats.OverFunction.Average
	case sortMaxCyclesOverFunction:
		if e.descending {
			return iStats.OverFunction.Max > jStats.OverFunction.Max
		}
		return iStats.OverFunction.Max < jStats.OverFunction.Max
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
	kernel     InKernel

	functionComparison bool
}

func (e SortedFunctions) Sort() {
	sort.Stable(e)
}

func (e *SortedFunctions) SetKernel(kernel InKernel) {
	e.kernel = kernel
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
	var iStats Stats
	var jStats Stats

	switch e.kernel {
	case InVBLANK:
		iStats = e.Functions[i].StatsVBLANK
		jStats = e.Functions[j].StatsVBLANK
	case InScreen:
		iStats = e.Functions[i].StatsScreen
		jStats = e.Functions[j].StatsScreen
	case InOverscan:
		iStats = e.Functions[i].StatsOverscan
		jStats = e.Functions[j].StatsOverscan
	default:
		iStats = e.Functions[i].Stats
		jStats = e.Functions[j].Stats
	}

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
			return iStats.OverSource.Frame > jStats.OverSource.Frame
		}
		return iStats.OverSource.Frame < jStats.OverSource.Frame
	case sortAverageCyclesOverSource:
		if e.descending {
			return iStats.OverSource.Average > jStats.OverSource.Average
		}
		return iStats.OverSource.Average < jStats.OverSource.Average
	case sortMaxCyclesOverSource:
		if e.descending {
			return iStats.OverSource.Max > jStats.OverSource.Max
		}
		return iStats.OverSource.Max < jStats.OverSource.Max
	}

	return false
}

func (e SortedFunctions) Swap(i int, j int) {
	e.Functions[i], e.Functions[j] = e.Functions[j], e.Functions[i]
}

type SortedVariableMethod int

const (
	SortVariableByName SortedVariableMethod = iota
	SortVariableByAddress
)

type SortedVariables struct {
	Variables  []*SourceVariable
	Method     SortedVariableMethod
	Descending bool
}

func (e *SortedVariables) SortByName(descending bool) {
	e.Descending = descending
	e.Method = SortVariableByName
	sort.Stable(e)
}

func (e *SortedVariables) SortByAddress(descending bool) {
	e.Descending = descending
	e.Method = SortVariableByAddress
	sort.Stable(e)
}

func (v SortedVariables) Len() int {
	return len(v.Variables)
}

func (v SortedVariables) Less(i int, j int) bool {
	switch v.Method {
	case SortVariableByName:
		if v.Descending {
			return strings.ToUpper(v.Variables[i].Name) > strings.ToUpper(v.Variables[j].Name)
		}
		return strings.ToUpper(v.Variables[i].Name) < strings.ToUpper(v.Variables[j].Name)
	case SortVariableByAddress:
		if v.Descending {
			return v.Variables[i].Address > v.Variables[j].Address
		}
		return v.Variables[i].Address < v.Variables[j].Address
	}
	return false
}

func (v SortedVariables) Swap(i int, j int) {
	v.Variables[i], v.Variables[j] = v.Variables[j], v.Variables[i]
}
