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
	kernel     KernelVCS

	// sort by raw cycle counts, rather than percentages
	rawCycleCounts bool
}

func (e SortedLines) Sort() {
	sort.Stable(e)
}

func (e *SortedLines) SetKernel(kernel KernelVCS) {
	e.kernel = kernel
}

func (e *SortedLines) UseRawCyclesCounts(use bool) {
	e.rawCycleCounts = use
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
	case KernelVBLANK:
		iStats = e.Lines[i].Stats.VBLANK
		jStats = e.Lines[j].Stats.VBLANK
	case KernelScreen:
		iStats = e.Lines[i].Stats.Screen
		jStats = e.Lines[j].Stats.Screen
	case KernelOverscan:
		iStats = e.Lines[i].Stats.Overscan
		jStats = e.Lines[j].Stats.Overscan
	default:
		iStats = e.Lines[i].Stats.Overall
		jStats = e.Lines[j].Stats.Overall
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
	default:
		if e.rawCycleCounts {
			switch e.method {
			case sortFrameCyclesOverSource:
				if e.descending {
					return iStats.OverSource.FrameCount > jStats.OverSource.FrameCount
				}
				return iStats.OverSource.FrameCount < jStats.OverSource.FrameCount
			case sortAverageCyclesOverSource:
				if e.descending {
					return iStats.OverSource.AverageCount > jStats.OverSource.AverageCount
				}
				return iStats.OverSource.AverageCount < jStats.OverSource.AverageCount
			case sortMaxCyclesOverSource:
				if e.descending {
					return iStats.OverSource.MaxCount > jStats.OverSource.MaxCount
				}
				return iStats.OverSource.MaxCount < jStats.OverSource.MaxCount
			case sortFrameCyclesOverFunction:
				if e.descending {
					return iStats.OverFunction.FrameCount > jStats.OverFunction.FrameCount
				}
				return iStats.OverFunction.FrameCount < jStats.OverFunction.FrameCount
			case sortAverageCyclesOverFunction:
				if e.descending {
					return iStats.OverFunction.AverageCount > jStats.OverFunction.AverageCount
				}
				return iStats.OverFunction.AverageCount < jStats.OverFunction.AverageCount
			case sortMaxCyclesOverFunction:
				if e.descending {
					return iStats.OverFunction.MaxCount > jStats.OverFunction.MaxCount
				}
				return iStats.OverFunction.MaxCount < jStats.OverFunction.MaxCount
			}
		} else {
			switch e.method {
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
			case sortFrameCyclesOverFunction:
				if e.descending {
					return iStats.OverFunction.Frame > jStats.OverFunction.Frame
				}
				return iStats.OverFunction.Frame < jStats.OverFunction.Frame
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
		}
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
	kernel     KernelVCS
	cumulative bool

	functionComparison bool

	// sort by raw cycle counts, rather than percentages
	rawCycleCounts bool
}

func (e SortedFunctions) Sort() {
	sort.Stable(e)
}

func (e *SortedFunctions) SetKernel(kernel KernelVCS) {
	e.kernel = kernel
}

func (e *SortedFunctions) SetCumulative(set bool) {
	e.cumulative = set
}

func (e *SortedFunctions) UseRawCyclesCounts(use bool) {
	e.rawCycleCounts = use
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
	case KernelVBLANK:
		if e.cumulative {
			iStats = e.Functions[i].CumulativeStats.VBLANK
			jStats = e.Functions[j].CumulativeStats.VBLANK
		} else {
			iStats = e.Functions[i].FlatStats.VBLANK
			jStats = e.Functions[j].FlatStats.VBLANK
		}
	case KernelScreen:
		if e.cumulative {
			iStats = e.Functions[i].CumulativeStats.Screen
			jStats = e.Functions[j].CumulativeStats.Screen
		} else {
			iStats = e.Functions[i].FlatStats.Screen
			jStats = e.Functions[j].FlatStats.Screen
		}
	case KernelOverscan:
		if e.cumulative {
			iStats = e.Functions[i].CumulativeStats.Overscan
			jStats = e.Functions[j].CumulativeStats.Overscan
		} else {
			iStats = e.Functions[i].FlatStats.Overscan
			jStats = e.Functions[j].FlatStats.Overscan
		}
	default:
		if e.cumulative {
			iStats = e.Functions[i].CumulativeStats.Overall
			jStats = e.Functions[j].CumulativeStats.Overall
		} else {
			iStats = e.Functions[i].FlatStats.Overall
			jStats = e.Functions[j].FlatStats.Overall
		}
	}

	switch e.method {
	case sortFile:
		// some functions don't have a declaration line
		if e.Functions[i].DeclLine == nil || e.Functions[j].DeclLine == nil {
			return true
		}

		if e.descending {
			return e.Functions[i].DeclLine.File.Filename > e.Functions[j].DeclLine.File.Filename
		}
		return e.Functions[i].DeclLine.File.Filename < e.Functions[j].DeclLine.File.Filename
	case sortFunction:
		if e.descending {
			return strings.ToUpper(e.Functions[i].Name) > strings.ToUpper(e.Functions[j].Name)
		}
		return strings.ToUpper(e.Functions[i].Name) < strings.ToUpper(e.Functions[j].Name)
	default:
		if e.rawCycleCounts {
			switch e.method {
			case sortFrameCyclesOverSource:
				if e.descending {
					return iStats.OverSource.FrameCount > jStats.OverSource.FrameCount
				}
				return iStats.OverSource.FrameCount < jStats.OverSource.FrameCount
			case sortAverageCyclesOverSource:
				if e.descending {
					return iStats.OverSource.AverageCount > jStats.OverSource.AverageCount
				}
				return iStats.OverSource.AverageCount < jStats.OverSource.AverageCount
			case sortMaxCyclesOverSource:
				if e.descending {
					return iStats.OverSource.MaxCount > jStats.OverSource.MaxCount
				}
				return iStats.OverSource.MaxCount < jStats.OverSource.MaxCount
			}
		} else {
			switch e.method {
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
		}
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
		ia, _ := v.Variables[i].Address()
		ja, _ := v.Variables[j].Address()
		if v.Descending {
			return ia > ja
		}
		return ia < ja
	}
	return false
}

func (v SortedVariables) Swap(i int, j int) {
	v.Variables[i], v.Variables[j] = v.Variables[j], v.Variables[i]
}

// SortedVariabelsLocal is exactly the same as the SortedVariables type except
// for the type of the Variables field. this is a good candidate for replacing
// with a Go1.19 generic solution
type SortedVariablesLocal struct {
	Variables  []*sourceVariableLocal
	Method     SortedVariableMethod
	Descending bool
}

func (e *SortedVariablesLocal) SortByName(descending bool) {
	e.Descending = descending
	e.Method = SortVariableByName
	sort.Stable(e)
}

func (e *SortedVariablesLocal) SortByAddress(descending bool) {
	e.Descending = descending
	e.Method = SortVariableByAddress
	sort.Stable(e)
}

func (v SortedVariablesLocal) Len() int {
	return len(v.Variables)
}

func (v SortedVariablesLocal) Less(i int, j int) bool {
	switch v.Method {
	case SortVariableByName:
		if v.Descending {
			return strings.ToUpper(v.Variables[i].Name) > strings.ToUpper(v.Variables[j].Name)
		}
		return strings.ToUpper(v.Variables[i].Name) < strings.ToUpper(v.Variables[j].Name)
	case SortVariableByAddress:
		ia, _ := v.Variables[i].Address()
		ja, _ := v.Variables[j].Address()
		if v.Descending {
			return ia > ja
		}
		return ia < ja
	}
	return false
}

func (v SortedVariablesLocal) Swap(i int, j int) {
	v.Variables[i], v.Variables[j] = v.Variables[j], v.Variables[i]
}
