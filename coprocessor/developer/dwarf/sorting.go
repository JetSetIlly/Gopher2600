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
	"sort"
	"strings"

	"github.com/jetsetilly/gopher2600/coprocessor/developer/profiling"
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
	sortFrameCalls
	sortAverageCalls
	sortMaxCalls
)

type SortedLines struct {
	Lines      []*SourceLine
	method     sortMethods
	descending bool
	kernel     profiling.Focus

	// sort by raw cycle counts, rather than percentages
	rawCycleCounts bool
}

func (e SortedLines) Sort() {
	sort.Stable(e)
}

func (e *SortedLines) SetKernel(kernel profiling.Focus) {
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
	var iStats profiling.Cycles
	var jStats profiling.Cycles

	switch e.kernel {
	case profiling.FocusVBLANK:
		iStats = e.Lines[i].Stats.VBLANK
		jStats = e.Lines[j].Stats.VBLANK
	case profiling.FocusScreen:
		iStats = e.Lines[i].Stats.Screen
		jStats = e.Lines[j].Stats.Screen
	case profiling.FocusOverscan:
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
					return iStats.CyclesProgram.FrameCount > jStats.CyclesProgram.FrameCount
				}
				return iStats.CyclesProgram.FrameCount < jStats.CyclesProgram.FrameCount
			case sortAverageCyclesOverSource:
				if e.descending {
					return iStats.CyclesProgram.AverageCount > jStats.CyclesProgram.AverageCount
				}
				return iStats.CyclesProgram.AverageCount < jStats.CyclesProgram.AverageCount
			case sortMaxCyclesOverSource:
				if e.descending {
					return iStats.CyclesProgram.MaxCount > jStats.CyclesProgram.MaxCount
				}
				return iStats.CyclesProgram.MaxCount < jStats.CyclesProgram.MaxCount
			case sortFrameCyclesOverFunction:
				if e.descending {
					return iStats.CyclesFunction.FrameCount > jStats.CyclesFunction.FrameCount
				}
				return iStats.CyclesFunction.FrameCount < jStats.CyclesFunction.FrameCount
			case sortAverageCyclesOverFunction:
				if e.descending {
					return iStats.CyclesFunction.AverageCount > jStats.CyclesFunction.AverageCount
				}
				return iStats.CyclesFunction.AverageCount < jStats.CyclesFunction.AverageCount
			case sortMaxCyclesOverFunction:
				if e.descending {
					return iStats.CyclesFunction.MaxCount > jStats.CyclesFunction.MaxCount
				}
				return iStats.CyclesFunction.MaxCount < jStats.CyclesFunction.MaxCount
			}
		} else {
			switch e.method {
			case sortFrameCyclesOverSource:
				if e.descending {
					return iStats.CyclesProgram.FrameCount > jStats.CyclesProgram.FrameCount
				}
				return iStats.CyclesProgram.FrameCount < jStats.CyclesProgram.FrameCount
			case sortAverageCyclesOverSource:
				if e.descending {
					return iStats.CyclesProgram.AverageCount > jStats.CyclesProgram.AverageCount
				}
				return iStats.CyclesProgram.AverageCount < jStats.CyclesProgram.AverageCount
			case sortMaxCyclesOverSource:
				if e.descending {
					return iStats.CyclesProgram.MaxCount > jStats.CyclesProgram.MaxCount
				}
				return iStats.CyclesProgram.MaxCount < jStats.CyclesProgram.MaxCount
			case sortFrameCyclesOverFunction:
				if e.descending {
					return iStats.CyclesFunction.FrameCount > jStats.CyclesFunction.FrameCount
				}
				return iStats.CyclesFunction.FrameCount < jStats.CyclesFunction.FrameCount
			case sortAverageCyclesOverFunction:
				if e.descending {
					return iStats.CyclesFunction.AverageCount > jStats.CyclesFunction.AverageCount
				}
				return iStats.CyclesFunction.AverageCount < jStats.CyclesFunction.AverageCount
			case sortMaxCyclesOverFunction:
				if e.descending {
					return iStats.CyclesFunction.MaxCount > jStats.CyclesFunction.MaxCount
				}
				return iStats.CyclesFunction.MaxCount < jStats.CyclesFunction.MaxCount
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
	focus      profiling.Focus
	cumulative bool

	functionComparison bool

	// sort by raw cycle counts, rather than percentages
	rawCycleCounts bool

	// parameter field can be used to pass additional information to a sort method
	parameter any
}

func (e SortedFunctions) Sort() {
	sort.Stable(e)
}

func (e *SortedFunctions) SetFocus(focus profiling.Focus) {
	e.focus = focus
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

func (e *SortedFunctions) SortByFrameCalls(descending bool) {
	e.descending = descending
	e.method = sortFrameCalls
	sort.Stable(e)
}

func (e *SortedFunctions) SortByAverageCalls(descending bool) {
	e.descending = descending
	e.method = sortAverageCalls
	sort.Stable(e)
}

func (e *SortedFunctions) SortByMaxCalls(descending bool) {
	e.descending = descending
	e.method = sortMaxCalls
	sort.Stable(e)
}

func (e SortedFunctions) Len() int {
	return len(e.Functions)
}

func (e SortedFunctions) Less(i int, j int) bool {
	var a profiling.Cycles
	var b profiling.Cycles
	var na profiling.Calls
	var nb profiling.Calls

	switch e.focus {
	case profiling.FocusVBLANK:
		if e.cumulative {
			a = e.Functions[i].CumulativeCycles.VBLANK
			b = e.Functions[j].CumulativeCycles.VBLANK
		} else {
			a = e.Functions[i].FlatCycles.VBLANK
			b = e.Functions[j].FlatCycles.VBLANK
		}
		na = e.Functions[i].NumCalls.VBLANK
		nb = e.Functions[j].NumCalls.VBLANK
	case profiling.FocusScreen:
		if e.cumulative {
			a = e.Functions[i].CumulativeCycles.Screen
			b = e.Functions[j].CumulativeCycles.Screen
		} else {
			a = e.Functions[i].FlatCycles.Screen
			b = e.Functions[j].FlatCycles.Screen
		}
		na = e.Functions[i].NumCalls.Screen
		nb = e.Functions[j].NumCalls.Screen
	case profiling.FocusOverscan:
		if e.cumulative {
			a = e.Functions[i].CumulativeCycles.Overscan
			b = e.Functions[j].CumulativeCycles.Overscan
		} else {
			a = e.Functions[i].FlatCycles.Overscan
			b = e.Functions[j].FlatCycles.Overscan
		}
		na = e.Functions[i].NumCalls.Overscan
		nb = e.Functions[j].NumCalls.Overscan
	default:
		if e.cumulative {
			a = e.Functions[i].CumulativeCycles.Overall
			b = e.Functions[j].CumulativeCycles.Overall
		} else {
			a = e.Functions[i].FlatCycles.Overall
			b = e.Functions[j].FlatCycles.Overall
		}
		na = e.Functions[i].NumCalls.Overall
		nb = e.Functions[j].NumCalls.Overall
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
	case sortFrameCalls:
		if e.descending {
			return na.FrameCount > nb.FrameCount
		}
		return na.FrameCount < nb.FrameCount
	case sortAverageCalls:
		if e.descending {
			return na.AverageCount > nb.AverageCount
		}
		return na.AverageCount < nb.AverageCount
	case sortMaxCalls:
		if e.descending {
			return na.MaxCount > nb.MaxCount
		}
		return na.MaxCount < nb.MaxCount
	default:
		if e.rawCycleCounts {
			switch e.method {
			case sortFrameCyclesOverSource:
				if e.descending {
					return a.CyclesProgram.FrameCount > b.CyclesProgram.FrameCount
				}
				return a.CyclesProgram.FrameCount < b.CyclesProgram.FrameCount
			case sortAverageCyclesOverSource:
				if e.descending {
					return a.CyclesProgram.AverageCount > b.CyclesProgram.AverageCount
				}
				return a.CyclesProgram.AverageCount < b.CyclesProgram.AverageCount
			case sortMaxCyclesOverSource:
				if e.descending {
					return a.CyclesProgram.MaxCount > b.CyclesProgram.MaxCount
				}
				return a.CyclesProgram.MaxCount < b.CyclesProgram.MaxCount
			}
		} else {
			switch e.method {
			case sortFrameCyclesOverSource:
				if e.descending {
					return a.CyclesProgram.FrameCount > b.CyclesProgram.FrameCount
				}
				return a.CyclesProgram.FrameCount < b.CyclesProgram.FrameCount
			case sortAverageCyclesOverSource:
				if e.descending {
					return a.CyclesProgram.AverageCount > b.CyclesProgram.AverageCount
				}
				return a.CyclesProgram.AverageCount < b.CyclesProgram.AverageCount
			case sortMaxCyclesOverSource:
				if e.descending {
					return a.CyclesProgram.MaxCount > b.CyclesProgram.MaxCount
				}
				return a.CyclesProgram.MaxCount < b.CyclesProgram.MaxCount
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
	Locals     []*SourceVariableLocal
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
	return len(v.Locals)
}

func (v SortedVariablesLocal) Less(i int, j int) bool {
	switch v.Method {
	case SortVariableByName:
		if v.Descending {
			return strings.ToUpper(v.Locals[i].Name) > strings.ToUpper(v.Locals[j].Name)
		}
		return strings.ToUpper(v.Locals[i].Name) < strings.ToUpper(v.Locals[j].Name)
	case SortVariableByAddress:
		ia, _ := v.Locals[i].Address()
		ja, _ := v.Locals[j].Address()
		if v.Descending {
			return ia > ja
		}
		return ia < ja
	}
	return false
}

func (v SortedVariablesLocal) Swap(i int, j int) {
	v.Locals[i], v.Locals[j] = v.Locals[j], v.Locals[i]
}
