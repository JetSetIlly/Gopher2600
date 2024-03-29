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

// Package reflection monitors the emulated hardware for conditions that would
// otherwise not be visible through normal emulation. The reflection system
// must be stepped every video cycle if all information is to be gathered.
//
// Note that with regards to the debugger package, Step() is called manually
// from the input loop as appropriate. However, NewFrame() is handled by the
// television - the Reflector having been added as a FrameTrigger.
package reflection
