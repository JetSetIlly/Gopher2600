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
// For quick disassemblies the FromCartridge() function can be used.  Debuggers
// will probably find it more useful however, to disassemble from the memory of
// an already instantiated VCS.
//
//	disasm, _ := disassembly.FromMemory(cartMem, symbols.NewTable())
//
// The FromMemory() function takes an instance of a symbols.Table or nil. In
// the example above, the result of NewTable() has been used, which is fine but
// limits the potential of the disassembly package. For best results, the
// symbols.ReadSymbolsFile() function should be used (see symbols package for
// details). Note that the FromCartridge() function handles symbols files for
// you.
//
// The Write() group of functions "print" disassambly entries of type
// EntryTypeDecode only. Useful for printing static disassemblies of
// a cartridge but probably not much else.
//
// The iteration types provides a convenient way of iterating of the
// disassembly entries. It takes care of empty entries and entries not of the
// correct entry type.
//
// The Grep() function provides a quick way of searching the disassembly with a
// scope directive. More complex search schemes can be written with the
// iteration types.
package disassembly
