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

// SetFunctionFilter to function name. Returns false if filter is empty.
func (src *Source) SetFunctionFilter(function string) bool {
	src.FunctionFilter = function
	src.FunctionFilteredLines.Lines = src.FunctionFilteredLines.Lines[:0]

	if !src.HasFunctionFilter() {
		return false
	}

	for _, ln := range src.SortedLines.Lines {
		if ln.Function.Name == src.FunctionFilter {
			src.FunctionFilteredLines.Lines = append(src.FunctionFilteredLines.Lines, ln)
		}
	}
	src.FunctionFilteredLines.SortByLineAndFunction(false)

	return len(src.FunctionFilteredLines.Lines) != 0
}

// DropFunctionFilter drops the existing filter.
func (src *Source) DropFunctionFilter() {
	src.FunctionFilter = ""
	src.FunctionFilteredLines.Lines = src.FunctionFilteredLines.Lines[:0]
}

// HasFunctionFilter returns true if filter is set.
func (src *Source) HasFunctionFilter() bool {
	return src.FunctionFilter != ""
}
