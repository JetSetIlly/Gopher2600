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

package prefs

// list of preference values that are no longer used.
var defunct = []string{
	"debugger.randpins",
	"debugger.randstate",
}

// returns true if string is in list of defunct values.
func isDefunct(s string) bool {
	for _, m := range defunct {
		if s == m {
			return true
		}
	}
	return false
}
