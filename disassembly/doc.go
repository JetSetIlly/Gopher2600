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

// Package disassembly coordinates the disassembly of Atari2600 (6507)
// cartridges.
//
// Should not be confused with the disassembly sub-package that is found in the
// coprocessor package.
//
// For quick disassemblies the FromCartridge() function can be used.  Debuggers
// will probably find it more useful however, to disassemble from the memory of
// an already instantiated VCS.
//
// Disassemblies will be performed in the background. This helps with startup
// speed but does mean that the display of disassembly may be delayed. Emulation
// can go ahead while disassembly is being performed and the results merged with
// disassembly once it has completed.
package disassembly
