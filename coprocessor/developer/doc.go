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

// Package developer offers additional functionality to the developer of ROMs
// that use a coprocessor. For instance, it handles the loading of .map and
// .obj files if they have been generated during the compilation of the 2600
// ROM. The .map and .obj files are used to provide source code level
// information during execution.
//
// Objdump type is a very basic parser for obj files as produced by "objdump
// -S" on the base elf file that is used to create a cartridge binary
package developer
