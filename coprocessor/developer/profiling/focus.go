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

package profiling

// Focus values are used to indicate when (very broadly) a function, a line of
// code, etc. has executed
type Focus int

// List of Focus values
const (
	FocusAll      Focus = 0x00
	FocusScreen   Focus = 0x01
	FocusVBLANK   Focus = 0x02
	FocusOverscan Focus = 0x04
)

func (k Focus) String() string {
	switch k {
	case FocusScreen:
		return "Screen"
	case FocusVBLANK:
		return "VBLANK"
	case FocusOverscan:
		return "Overscan"
	}
	return "All"
}

// List of Focus values as strings
var FocusOptions = []string{"All", "VBLANK", "Screen", "Overscan"}
