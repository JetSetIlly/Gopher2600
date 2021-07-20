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

// Package terminal defines the operations required for command-line
// interaction with the debugger.
//
// For flexibility, terminal interaction happens through the Terminal
// interface. There are two reference implementations of this interface: the
// PlainTerminal and the ColorTerminal, found respectively in the plainterm and
// colorterm sub-packages.
//
// Note that history is not handled by this package - an implementation must
// implement this itself. Of the two reference implementations, the
// ColorTerminal package provides an example.
//
// TabCompletion is handled by the commandline package if required. Again, the
// ColorTerminal implementation is a good example of how to use this package.
package terminal
