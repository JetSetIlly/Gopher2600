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

// SortLinesMethod specifies the sort method to be applied when sorting the
// entries in the SortedLines type
type SortLinesMethod int

// List of valid SortLinesMethod values
const (
	SortLinesFile SortLinesMethod = iota
	SortLinesFunction
	SortLinesNumber
	SortLinesFrameCycles
	SortLinesAverageCycles
	SortLinesMaxCycles
)

// SortedLines holds the list of SourceLines sorted by the specfied method
type SortedLines struct {
	Lines      []*SourceLine
	method     SortLinesMethod
	descending bool
	focus      profiling.Focus

	// overProgram and load are only applicable to sort methods that work on the
	// number of cycles
	overProgram bool
	load        bool
}

// Sort is a stable sort of the Lines in the SortedLines type
//
// The overProgram and load parameters only apply to sort methods that work on
// the number of cycles
func (e *SortedLines) Sort(method SortLinesMethod, overProgram bool, load bool, descending bool, focus profiling.Focus) {
	e.method = method
	e.overProgram = overProgram
	e.load = load
	e.descending = descending
	e.focus = focus
	sort.Stable(e)
}

func (e SortedLines) Len() int {
	return len(e.Lines)
}

func (e SortedLines) Less(i int, j int) bool {
	var af profiling.CyclesScope
	var bf profiling.CyclesScope

	switch e.focus {
	case profiling.FocusVBLANK:
		af = e.Lines[i].Cycles.VBLANK
		bf = e.Lines[j].Cycles.VBLANK
	case profiling.FocusScreen:
		af = e.Lines[i].Cycles.Screen
		bf = e.Lines[j].Cycles.Screen
	case profiling.FocusOverscan:
		af = e.Lines[i].Cycles.Overscan
		bf = e.Lines[j].Cycles.Overscan
	default:
		af = e.Lines[i].Cycles.Overall
		bf = e.Lines[j].Cycles.Overall
	}

	var al profiling.CycleFigures
	var bl profiling.CycleFigures

	if e.overProgram {
		al = af.CyclesProgram
		bl = bf.CyclesProgram
	} else {
		al = af.CyclesFunction
		bl = bf.CyclesFunction
	}

	switch e.method {
	case SortLinesFile:
		return e.Lines[i].Function.DeclLine.File.Filename <= e.Lines[j].Function.DeclLine.File.Filename != e.descending
	case SortLinesFunction:
		return strings.ToUpper(e.Lines[i].Function.Name) <= strings.ToUpper(e.Lines[j].Function.Name) != e.descending
	case SortLinesNumber:
		return e.Lines[i].LineNumber <= e.Lines[j].LineNumber != e.descending
	case SortLinesFrameCycles:
		if e.load {
			return al.FrameLoad <= bl.FrameLoad != e.descending
		}
		return al.FrameCount <= bl.FrameCount != e.descending
	case SortLinesAverageCycles:
		if e.load {
			return al.AverageLoad <= bl.AverageLoad != e.descending
		}
		return al.AverageCount <= bl.AverageCount != e.descending
	case SortLinesMaxCycles:
		if e.load {
			return al.MaxLoad <= bl.MaxLoad != e.descending
		}
		return al.MaxCount <= bl.MaxCount != e.descending
	}

	return false
}

func (e SortedLines) Swap(i int, j int) {
	e.Lines[i], e.Lines[j] = e.Lines[j], e.Lines[i]
}

// SortFunctionsMethod specifies the sort method to be applied when sorting the
// entries in the SortedFunctions type
type SortFunctionsMethod int

// List of valid SortFunctionsMethod values
const (
	SortFunctionsFile SortFunctionsMethod = iota
	SortFunctionsName
	SortFunctionsFrameCycles
	SortFunctionsAverageCycles
	SortFunctionsMaxCycles
	SortFunctionsFrameCalls
	SortFunctionsAverageCalls
	SortFunctionsMaxCalls
	SortFunctionsFrameCyclesPerCall
	SortFunctionsAverageCyclesPerCall
	SortFunctionsMaxCyclesPerCall
)

// SortedFunctions holds the list of SourceFunction sorted by the specfied method
type SortedFunctions struct {
	Functions  []*SourceFunction
	method     SortFunctionsMethod
	descending bool
	focus      profiling.Focus

	// cumulative and load are only applicable to sort methods that work on the
	// number of cycles
	cumulative bool
	load       bool
}

// Sort is a stable sort of the Functions in the SortedFunctions type
//
// The cumulative and load parameters only apply to sort methods that work on
// the number of cycles
func (e *SortedFunctions) Sort(method SortFunctionsMethod, cumulative bool, load bool, descending bool, focus profiling.Focus) {
	e.method = method
	e.cumulative = cumulative
	e.load = load
	e.descending = descending
	e.focus = focus
	sort.Stable(e)
}

// Len implements the sort.Interface
func (e SortedFunctions) Len() int {
	return len(e.Functions)
}

// Less implements the sort.Interface
func (e SortedFunctions) Less(i int, j int) bool {
	var as profiling.Cycles
	var bs profiling.Cycles

	if e.cumulative {
		as = e.Functions[i].CumulativeCycles
		bs = e.Functions[j].CumulativeCycles
	} else {
		as = e.Functions[i].Cycles
		bs = e.Functions[j].Cycles
	}

	var af profiling.CyclesScope
	var bf profiling.CyclesScope
	var afc profiling.CallsScope
	var bfc profiling.CallsScope
	var afp profiling.CyclesPerCallScope
	var bfp profiling.CyclesPerCallScope

	switch e.focus {
	case profiling.FocusVBLANK:
		af = as.VBLANK
		bf = bs.VBLANK
		afc = e.Functions[i].NumCalls.VBLANK
		bfc = e.Functions[j].NumCalls.VBLANK
		afp = e.Functions[i].CyclesPerCall.VBLANK
		bfp = e.Functions[j].CyclesPerCall.VBLANK
	case profiling.FocusScreen:
		af = as.Screen
		bf = bs.Screen
		afc = e.Functions[i].NumCalls.Screen
		bfc = e.Functions[j].NumCalls.Screen
		afp = e.Functions[i].CyclesPerCall.Screen
		bfp = e.Functions[j].CyclesPerCall.Screen
	case profiling.FocusOverscan:
		af = as.Overscan
		bf = bs.Overscan
		afc = e.Functions[i].NumCalls.Overscan
		bfc = e.Functions[j].NumCalls.Overscan
		afp = e.Functions[i].CyclesPerCall.Overscan
		bfp = e.Functions[j].CyclesPerCall.Overscan
	default:
		af = as.Overall
		bf = bs.Overall
		afc = e.Functions[i].NumCalls.Overall
		bfc = e.Functions[j].NumCalls.Overall
		afp = e.Functions[i].CyclesPerCall.Overall
		bfp = e.Functions[j].CyclesPerCall.Overall
	}

	switch e.method {
	case SortFunctionsFile:
		// some functions don't have a declaration line
		if e.Functions[i].DeclLine == nil || e.Functions[j].DeclLine == nil {
			return true
		}
		return (e.Functions[i].DeclLine.File.Filename <= e.Functions[j].DeclLine.File.Filename) != e.descending
	case SortFunctionsName:
		return (strings.ToUpper(e.Functions[i].Name) <= strings.ToUpper(e.Functions[j].Name)) != e.descending
	case SortFunctionsFrameCycles:
		if e.load {
			return af.CyclesProgram.FrameLoad <= bf.CyclesProgram.FrameLoad != e.descending
		}
		return af.CyclesProgram.FrameCount <= bf.CyclesProgram.FrameCount != e.descending
	case SortFunctionsAverageCycles:
		if e.load {
			return af.CyclesProgram.AverageLoad <= bf.CyclesProgram.AverageLoad != e.descending
		}
		return af.CyclesProgram.AverageCount <= bf.CyclesProgram.AverageCount != e.descending
	case SortFunctionsMaxCycles:
		if e.load {
			return af.CyclesProgram.MaxLoad <= bf.CyclesProgram.MaxLoad != e.descending
		}
		return af.CyclesProgram.MaxCount <= bf.CyclesProgram.MaxCount != e.descending
	case SortFunctionsFrameCalls:
		return afc.FrameCount <= bfc.FrameCount != e.descending
	case SortFunctionsAverageCalls:
		return afc.AverageCount <= bfc.AverageCount != e.descending
	case SortFunctionsMaxCalls:
		return afc.MaxCount <= bfc.MaxCount != e.descending
	case SortFunctionsFrameCyclesPerCall:
		return afp.FrameCount <= bfp.FrameCount != e.descending
	case SortFunctionsAverageCyclesPerCall:
		return afp.AverageCount <= bfp.AverageCount != e.descending
	case SortFunctionsMaxCyclesPerCall:
		return afp.MaxCount <= bfp.MaxCount != e.descending
	}

	return false
}

// Swap implements the sort.Interface
func (e SortedFunctions) Swap(i int, j int) {
	e.Functions[i], e.Functions[j] = e.Functions[j], e.Functions[i]
}

// SortVariablesMethod specifies the sort method to be applied when sorting the
// entries in the SortedVariables type
type SortVariablesMethod int

// List of valid SortVariablesMethod values
const (
	SortVariablesName SortVariablesMethod = iota
	SortVariablesAddress
)

// SortedVariables holds the list of SourceVariable sorted by the specfied method
type SortedVariables struct {
	Variables  []*SourceVariable
	method     SortVariablesMethod
	descending bool
}

// Sort is a stable sort of the Variables in the SortedVariables type
func (e *SortedVariables) Sort(method SortVariablesMethod, descending bool) {
	e.method = method
	e.descending = descending
	sort.Stable(e)
}

// Len implements the sort.Interface
func (v SortedVariables) Len() int {
	return len(v.Variables)
}

// Less implements the sort.Interface
func (v SortedVariables) Less(i int, j int) bool {
	switch v.method {
	case SortVariablesName:
		return strings.ToUpper(v.Variables[i].Name) <= strings.ToUpper(v.Variables[j].Name) != v.descending
	case SortVariablesAddress:
		ia, _ := v.Variables[i].Address()
		ja, _ := v.Variables[j].Address()
		return ia <= ja != v.descending
	}
	return false
}

// Swap implements the sort.Interface
func (v SortedVariables) Swap(i int, j int) {
	v.Variables[i], v.Variables[j] = v.Variables[j], v.Variables[i]
}

// SortedVariablesLocal holds the list of SourceVariableLocal sorted by the specfied method
//
// This is exactly the same type and implementation as SortedVariables. With a
// bit of work this could probably be improved with an variable interface that
// handles both SourceVariable and SourceVariableLocal
type SortedVariablesLocal struct {
	Variables  []*SourceVariableLocal
	method     SortVariablesMethod
	descending bool
}

// Sort is a stable sort of the Variables in the SortedVariablesLocal type
func (e *SortedVariablesLocal) Sort(method SortVariablesMethod, descending bool) {
	e.method = method
	e.descending = descending
	sort.Stable(e)
}

// Len implements the sort.Interface
func (v SortedVariablesLocal) Len() int {
	return len(v.Variables)
}

// Less implements the sort.Interface
func (v SortedVariablesLocal) Less(i int, j int) bool {
	switch v.method {
	case SortVariablesName:
		return strings.ToUpper(v.Variables[i].Name) < strings.ToUpper(v.Variables[j].Name) != v.descending
	case SortVariablesAddress:
		ia, _ := v.Variables[i].Address()
		ja, _ := v.Variables[j].Address()
		return ia <= ja != v.descending
	}
	return false
}

// Swap implements the sort.Interface
func (v SortedVariablesLocal) Swap(i int, j int) {
	v.Variables[i], v.Variables[j] = v.Variables[j], v.Variables[i]
}
