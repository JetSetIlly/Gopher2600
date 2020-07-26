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

// Package symbols helps keep track of address symbols. The primary structure
// for this is the Table type. There are two recommended ways of instantiating
// this type. NewTable() will create a table instance with the default or
// canonical Atari 2600 symbol names. For example, AUDC0 refers to the $15
// write address.
//
// The second and more flexible way of instantiating the symbols Table is with
// the ReadSymbolsFile() function. This function will try to read a symbols
// file for the named cartridge and parse its contents. It will fail silently
// if it cannot.
//
// ReadSymbolFile() will always give addresses the default or canonised symbol.
// In this way it is a superset of the NewTable() function.
package symbols
