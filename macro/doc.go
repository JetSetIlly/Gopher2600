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

// Package macro implements an input system that processes instructions from a
// macro script.
//
// The macro language is very simple and does not implement any flow control
// except basic loops.
//
//	DO loopCt [loopName]
//		...
//	LOOP
//
// The 'loopName' parameter is optional. When a loop is named the current
// counter value can be referenced as a variable in some contexts (currently,
// this is the SCREENSHOT instruction only).
//
// Loops can be nested.
//
// The WAIT instruction will pause the execution of the macro for the specified
// number of frames. If no value is given for this the number of frames defaults
// to 60.
//
// There are instructions that give basic control over the emulation (only left
// player joystick control and some panel operaitons).
//
//	LEFT, RIGHT, UP, DOWN, CENTRE, FIRE, NOFIRE, SELECT, RESET
//
// There is also an instruction to initiate a screenshot. The macro system is
// therefore useful to automate the collation of screenshots in a repeatable
// manner.
//
//	SCREENSHOT [filename suffix]
//
// The filename suffix parameter is optional. Without it the screenshot will be
// given the default but unique filename.
//
// If the filename suffix parameter is given then the name of the screenshot
// will be the name of the cartridge plus the suffix. Spaces will be replaced
// with underscores.
//
// In the context of the screenshot instruction, variables can referenced with
// the % symbol. For example, if a loop has been given the name "ct", then the
// following screenshot command could be written:
//
//	SCREENSHOT %ct
//
// Any errors in a macro script will result in a log entry and the termination
// of the macro execution.
//
// Lines can be commented by prefixing the line with two dashes (--). Leading
// and trailing white space is ignored.
package macro
