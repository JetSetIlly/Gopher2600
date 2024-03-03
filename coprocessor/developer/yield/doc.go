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

// Package yield records the current yielded state of a coprocessor. In
// particualar, it records the list of local variables visible at the current
// yield point
//
// It also handles the strobed viewed of the coprocessor and builds the strobed
// local variables list. Strobing allows the developer to see local variables
// when the emulation is running (see govern pacakge).
package yield
