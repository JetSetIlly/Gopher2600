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

import (
	"testing"

	"github.com/jetsetilly/gopher2600/test"
)

func testBreakpoints(t *testing.T, trm *mockTerm) {
	// debugger starts off with no breakpoints
	trm.command("LIST BREAKS")
	test.ExpectEquality(t, trm.lastLine(), "no breakpoints")

	// add a break. this should be successful so there should be no feedback
	trm.command("BREAK SL 100")
	test.ExpectEquality(t, trm.lastLine(), "")

	// list breaks and check last line of output
	trm.command("LIST BREAKS")
	test.ExpectEquality(t, trm.lastLine(), " 0: Scanline->100")

	// try to add same break. check error feedback
	trm.command("BREAK SL 100")
	test.ExpectEquality(t, trm.lastLine(), "already exists (Scanline->100)")

	// add multi-condition break
	trm.command("BREAK SL 100 & CL 100")
	test.ExpectEquality(t, trm.lastLine(), "")

	// check last line of list breaks. we've already added a break so this new
	// break should be number "1" (rather than number "0")
	trm.command("LIST BREAKS")
	test.ExpectEquality(t, trm.lastLine(), " 1: Scanline->100 & Clock->100")

	// try to add exactly the same breakpoint. expect failure
	trm.command("BREAK SL 100 & CL 100")
	test.ExpectEquality(t, trm.lastLine(), "already exists (Scanline->100 & Clock->100)")

	// as above but with alternative && connection
	trm.command("BREAK SL 100 && CL 100")
	test.ExpectEquality(t, trm.lastLine(), "already exists (Scanline->100 & Clock->100)")

	// the following break is logically the same as the previous break but expressed differently.
	// the debugger should not add it even though the expression is not exactly the same because of
	// the order of the AND statement.
	trm.command("BREAK CL 100 & SL 100")
	test.ExpectEquality(t, trm.lastLine(), "already exists (Clock->100 & Scanline->100)")

	// TOGGLE says to drop the breakpoint if it already exists
	trm.command("BREAK TOGGLE CL 100 & SL 100")
	test.ExpectEquality(t, trm.lastLine(), "")

	// the multi-line condition break has been removed (by toggling), leaving us with just the first
	// breakpoint we added
	trm.command("LIST BREAKS")
	test.ExpectEquality(t, trm.lastLine(), " 0: Scanline->100")

	// calling the BREAK TOGGLE command again adds the breakpoint if it doesn't exist.
	// we also check the last line of the output of LIST BREAKS
	trm.command("BREAK TOGGLE CL 100 & SL 100")
	test.ExpectEquality(t, trm.lastLine(), "")
	trm.command("LIST BREAKS")
	test.ExpectEquality(t, trm.lastLine(), " 1: Clock->100 & Scanline->100")

	// this is a different breakpoint
	trm.command("BREAK CL 100")
	test.ExpectEquality(t, trm.lastLine(), "")
	trm.command("LIST BREAKS")
	test.ExpectEquality(t, trm.lastLine(), " 2: Clock->100")

	// clear all breakpoints and check that the list is empty
	trm.command("CLEAR BREAKS")
	test.ExpectEquality(t, trm.lastLine(), "breakpoints cleared")
	trm.command("LIST BREAKS")
	test.ExpectEquality(t, trm.lastLine(), "no breakpoints")
}

func testBreakpoints_drop(t *testing.T, trm *mockTerm) {
	trm.command("BREAK DROP $1000")
	test.ExpectEquality(t, trm.lastLine(), "no such breakpoint (PC->0x1000)")

	trm.command("BREAK $1000")
	test.ExpectEquality(t, trm.lastLine(), "")

	trm.command("BREAK DROP $1000")
	test.ExpectEquality(t, trm.lastLine(), "")

	trm.command("LIST BREAKS")
	test.ExpectEquality(t, trm.lastLine(), "no breakpoints")

	trm.command("BREAK $1000")
	test.ExpectEquality(t, trm.lastLine(), "")
	trm.command("BREAK $3001")
	test.ExpectEquality(t, trm.lastLine(), "")

	trm.command("DROP BREAK 0")
	test.ExpectEquality(t, trm.lastLine(), "breakpoint #0 dropped")

	// the remaining breakpoint was set on 0x3001, which is a mirror of 0x1001.
	// breakpoints on PC addresses are always normalised so LIST BREAKS will show
	// the primary mirror address
	trm.command("LIST BREAKS")
	test.ExpectEquality(t, trm.lastLine(), " 0: PC->0x1001")
}
