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

// Package debugger implements a reaonably comprehensive debugging tool.
// Features include:
//
//	- cartridge disassembly
//	- memory peek and poke
//	- cpu and video cycle stepping
//	- basic scripting
//	- breakpoints
//	- traps
//	- watches
//
// Some of these features come courtesy of other packages, described elsewhere,
// and some are inherent in the gopher2600's emulation strategy, but all are
// nicely exposed via the debugger package.
//
// Initialisation of the debugger is done with the NewDebugger() function
//
//	dbg, _ := debugger.NewDebugger(television, gui, term)
//
// The tv, gui and term arguments must be instances of types that satisfy the
// repsective interfaces. This gives the debugger great flexibility and should
// allow easy porting to new platforms
//
// Interaction with the debugger is primarily through a terminal. The Terminal
// interface is defined in the terminal package. The colorterm and plainterm
// sub-packages provide good reference implementations.
//
// The GUI helps visualise the television and coordinates events (keyboard,
// mouse) which the debugger can then poll. A good reference implementation of
// a debugging GUI can be in found the gui.sdldebug package.
//
// The television argument should be an instance of TV. For all practical
// purposes this will be instance createed with television.NewTelevision(), but
// other implementations are possible if not yet available.
//
// Once initialised, the debugger can be started with the Start() function.
//
//	dbg.Start(initScript, cartloader)
//
// The initscript is a script previously created either by the script.Scribe
// package or by hand. The cartloader argument must be an instance of
// cartloader.
//
//
// Machine interaction with the debugger can be achieved through the terminal
// interface. For example, setting the debugging quantum can be done by
// returning a "QUANTUM CPU" string from a terminal.TermRead() implementation.
//
// Retrieving information from the debugger is more conveniently and more
// efficiently achieved with the Get*() commands. For example GetQuantum()
// returns the emulator's current quantum value.
package debugger
