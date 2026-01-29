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

func testWatches(t *testing.T, trm *mockTerm) {
	// debugger starts off with no watches
	trm.command("LIST WATCHES")
	test.ExpectEquality(t, trm.lastLine(), "no watches")

	// add read watch. there should be no output.
	trm.command("WATCH READ 0x80")
	test.ExpectEquality(t, trm.lastLine(), "")

	// try to re-add the same watch
	trm.command("WATCH READ 0x80")
	test.ExpectEquality(t, trm.lastLine(), "already being watched (0x0080 (RAM) read)")

	// list watches
	trm.command("LIST WATCHES")
	test.ExpectEquality(t, trm.lastLine(), " 0: 0x0080 (RAM) read")

	// try to re-add the same watch but with a different event selector
	trm.command("WATCH WRITE 0x80")
	test.ExpectEquality(t, trm.lastLine(), "")

	// list watches
	trm.command("LIST WATCHES")
	test.ExpectEquality(t, trm.lastLine(), " 1: 0x0080 (RAM) write")

	// clear watches
	trm.command("CLEAR WATCHES")
	test.ExpectEquality(t, trm.lastLine(), "watches cleared")

	// no watches after successful clear
	trm.command("LIST WATCHES")
	test.ExpectEquality(t, trm.lastLine(), "no watches")

	// try adding an invalid read address by symbol
	trm.command("WATCH READ VSYNC")
	test.ExpectEquality(t, trm.lastLine(), "invalid watch address (VSYNC) expecting 16-bit address or a read symbol")

	// add address by symbol. no read/write modifier means it tries
	trm.command("WATCH WRITE VSYNC")
	test.ExpectEquality(t, trm.lastLine(), "")

	// last item in list watches should be the new entry
	trm.command("LIST WATCHES")
	test.ExpectEquality(t, trm.lastLine(), " 0: 0x0000 (VSYNC) (TIA) write")

	// add address by symbol. no read/write modifier means it tries
	// plus a specific value
	trm.command("WATCH WRITE VSYNC 0x1")
	test.ExpectEquality(t, trm.lastLine(), "")

	// last item in list watches should be the new entry
	trm.command("LIST WATCHES")
	test.ExpectEquality(t, trm.lastLine(), " 1: 0x0000 (VSYNC) (TIA) write (value=0x01)")
}
