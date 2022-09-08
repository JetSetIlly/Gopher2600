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

type FunctionFilter struct {
	FunctionName string
	Function     *SourceFunction
	Lines        SortedLines
}

func (src *Source) AddFunctionFilter(functionName string) {
	for _, fn := range src.FunctionFilters {
		if fn.FunctionName == functionName {
			return
		}
	}

	ff := &FunctionFilter{
		FunctionName: functionName,
		Function:     src.Functions[functionName],
	}

	for _, ln := range src.SortedLines.Lines {
		if ln.Function.Name == ff.FunctionName {
			ff.Lines.Lines = append(ff.Lines.Lines, ln)
		}
	}

	src.FunctionFilters = append(src.FunctionFilters, ff)
}

// DropFunctionFilter drops the existing filter.
func (src *Source) DropFunctionFilter(functionName string) {
	for i, fn := range src.FunctionFilters {
		if fn.FunctionName == functionName {
			src.FunctionFilters = append(src.FunctionFilters[:i], src.FunctionFilters[i+1:]...)
			return
		}
	}
}
