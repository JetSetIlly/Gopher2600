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

package crunched

// Data provides the interface to a crunched data type
type Data interface {
	// IsCrunched returns true if data is currently crunched
	IsCrunched() bool

	// Size returns the uncrunched size and the current size of the data. If the
	// data is currently crunched then the two values will be the same
	Size() (int, int)

	// Data returns a pointer to the uncrunched data
	Data() *[]byte

	// Snapshot makes a copy of the data and crunching it if required. The data will
	// be uncrunched automatically when Data() function is called
	Snapshot() Data
}

// Inspection provides the interface to the crunched data type and provides the
// ability to inspect the data in its current form
type Inspection interface {
	Data

	// Inspect returns data in the current state. In other words, the data will
	// not be decrunched as it would be with the Data() function
	Inspect() *[]byte
}
