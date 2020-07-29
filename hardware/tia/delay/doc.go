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

// Package delay is a replacement for the future package, which has now been
// removed. This package (and the package it replaces) helps emulate the
// latching delays of the TIA.
//
// The future package served it's purpose during early, exploratory phases of
// the emulator's development. I wasn't sure at first what was needed and the
// future package developed as a way of supporting experimentation with the
// various elements of the TIA system.
//
// The delay package is a lot simpler and consequently a lot more efficient.
//
// The only element in the package is the Event type. An Event type instance
// represents a single future change to the TIA system, which will take place
// after the stated number of cycles.
//
// To effectively emulate the electronics of the TIA these Events can be
// dropped, rescheduled or premepted almost at will.
//
// Ordering of Events is rarely significant but in the instances where it is
// comments are included in the code.
package delay
