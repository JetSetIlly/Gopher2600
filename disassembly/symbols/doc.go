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

// Package symbols helps keep track of address symbols for the currently loaded
// cartridge. It will load symbols from a DASM symbol file if one can be found
// and will use standard (or canonical) symbol names as appropriate.
//
// In the context of the Gopher2600 project, it works best if the Symbol type
// is declared staticially and the ReadSymbolsFile() function called to
// populate the symbol tables. See the disassembly package for more details.
package symbols
