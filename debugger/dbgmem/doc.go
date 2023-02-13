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

// Package dbgmem sits between the debugger and the acutal VCS memory. In the
// context of the debugger it is more useful to address memory via this package
// rather than using the memory package directly.
//
// The key type provided by the package is the AddressInfo type. This type
// provides every detail about a memory address that you could want.
//
// The other key type is the MemoryDebug type. Initialise this is the usual way
// by specifying a VCS instance and symbols table. Note that both should be
// initialised and not left to point to nil - no checks are made in the dbgmem
// package.
//
// The MapAddress() function is the basic way you create an AddressInfo type.
// Specify whether the address is read or write address and the function does
// all the work. The address can even be a symbol, in which case the symbols
// table is consulted.
//
// The Peek() and Poke() functions complement the Peek() and Poke() functions
// in the memory package. In the dbgmem package case, Peek() and Poke() accept
// symbols as well as numeric addresses.
//
// Peek() and Poke() will return sentinal errors (PeekError and PokeError
// respectively) if a bus.AddressError is encountered).
package dbgmem
