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

package terminal

// Style is used to identify the category of text being sent to the
// Terminal.TermPrint() function. The terminal implementation can interpret
// this how it sees fit - the most likely treatment is to print different
// styles in different colours.
type Style int

// List of terminal styles.
const (
	// input from the user being echoed back to the user. echoed input has been
	// "normalised" (eg. capitalised, leading space removed, etc.)
	StyleEcho Style = iota

	// information from the internal help system
	StyleHelp

	// information from a command
	StyleFeedback

	// secondary information from a command
	StyleFeedbackSecondary

	// disassembly output for CPU instruction boundaries
	StyleInstructionStep

	// disassembly output for non-CPU instruction boundaries
	StyleSubStep

	// information about the machine
	StyleInstrument

	// information as a result of an error. errors can be generated by the
	// emulation or the debugger
	StyleError

	// information from the internal logging system
	StyleLog
)
