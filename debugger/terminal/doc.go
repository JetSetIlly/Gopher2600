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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

// Package terminal defines the operations required for user interaction with
// the debugger. While an implementation of the GUI interface, found in the GUI
// package, may allow some interaction with the debugger (eg. visually setting
// a breakpoint) the principle means of interaction with the debugger is the
// terminal.
//
// For flexibility, terminal interaction happens through the Terminal language
// interface. There are two reference implementations of this interface: the
// PlainTerminal and the ColorTerminal, found respectively in the plainterm and
// colorterm sub-packages.
package terminal
