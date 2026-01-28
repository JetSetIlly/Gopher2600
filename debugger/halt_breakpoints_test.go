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

package debugger_test

func (trm *mockTerm) testBreakpoints() {
	// debugger starts off with no breakpoints
	trm.sndInput("LIST BREAKS")
	trm.cmpOutput("no breakpoints")

	// add a break. this should be successful so there should be no feedback
	trm.sndInput("BREAK SL 100")
	trm.cmpOutput("")

	// list breaks and check last line of output
	trm.sndInput("LIST BREAKS")
	trm.cmpOutput(" 0: Scanline->100")

	// try to add same break. check error feedback
	trm.sndInput("BREAK SL 100")
	trm.cmpOutput("already exists (Scanline->100)")

	// add multi-condition break
	trm.sndInput("BREAK SL 100 & CL 100")
	trm.cmpOutput("")

	// check last line of list breaks. we've already added a break so this new
	// break should be number "1" (rather than number "0")
	trm.sndInput("LIST BREAKS")
	trm.cmpOutput(" 1: Scanline->100 & Clock->100")

	// try to add exactly the same breakpoint. expect failure
	trm.sndInput("BREAK SL 100 & CL 100")
	trm.cmpOutput("already exists (Scanline->100 & Clock->100)")

	// as above but with alternative && connection
	trm.sndInput("BREAK SL 100 && CL 100")
	trm.cmpOutput("already exists (Scanline->100 & Clock->100)")

	// the following break is logically the same as the previous break but expressed differently.
	// the debugger should not add it even though the expression is not exactly the same because of
	// the order of the AND statement.
	trm.sndInput("BREAK CL 100 & SL 100")
	trm.cmpOutput("already exists (Scanline->100 & Clock->100)")

	// TOGGLE says to drop the breakpoint if it already exists
	trm.sndInput("BREAK TOGGLE CL 100 & SL 100")
	trm.cmpOutput("")

	// the multi-line condition break has been removed (by toggling), leaving us with just the first
	// breakpoint we added
	trm.sndInput("LIST BREAKS")
	trm.cmpOutput(" 0: Scanline->100")

	// calling the BREAK TOGGLE command again adds the breakpoint if it doesn't exist.
	// we also check the last line of the output of LIST BREAKS
	trm.sndInput("BREAK TOGGLE CL 100 & SL 100")
	trm.sndInput("LIST BREAKS")
	trm.cmpOutput(" 1: Clock->100 & Scanline->100")

	// this is a different breakpoint
	trm.sndInput("BREAK CL 100")
	trm.cmpOutput("")
	trm.sndInput("LIST BREAKS")
	trm.cmpOutput(" 2: Clock->100")
}
