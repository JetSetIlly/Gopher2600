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
// removed.
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
// Because of how the TIA is constructed these Events can be dropped,
// rescheduled or premepted almost at will. Study of the TIA, and video/sprite
// sub-systems will show how the Events interact.
//
// The major difference between the delay package and the erstwhile future
// package is that the latter impliciely encoded the time relationship between
// two events but after the experimentation phase, it was found that this
// wasn't really necessary, except in a couple of very specific and easily
// mititgated instances.
package delay
