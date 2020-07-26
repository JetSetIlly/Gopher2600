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

// Package riot (RIOT) represents the active part of the PIA 6532. It does not
// handle the RAM part of the 6532, that can be found in the memory package.
//
// The active parts of the RIOT are:
//
//	Timer
//	I/O system
//
// The timer can be found in the timer package, whereas the I/O system can be
// found in the input package.
package riot
