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

package logger_test

import (
	"testing"

	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/test"
)

func TestLogger(t *testing.T) {
	tw := &test.Writer{}

	logger.Write(tw)
	test.Equate(t, tw.Compare(""), true)

	logger.Log("test", "this is a test")
	logger.Write(tw)
	test.Equate(t, tw.Compare("test: this is a test\n"), true)

	// clear the test.Writer buffer before continuing, makes comparisons easier
	// to manage
	tw.Clear()

	logger.Log("test2", "this is another test")
	logger.Write(tw)
	test.Equate(t, tw.Compare("test: this is a test\ntest2: this is another test\n"), true)

	// asking for too many entries in a Tail() should be okay
	tw.Clear()
	logger.Tail(tw, 100)
	test.Equate(t, tw.Compare("test: this is a test\ntest2: this is another test\n"), true)

	// asking for exactly the correct number of entries is okay
	tw.Clear()
	logger.Tail(tw, 2)
	test.Equate(t, tw.Compare("test: this is a test\ntest2: this is another test\n"), true)

	// asking for fewer entries is okay too
	tw.Clear()
	logger.Tail(tw, 1)
	test.Equate(t, tw.Compare("test2: this is another test\n"), true)

	// and no entries
	tw.Clear()
	logger.Tail(tw, 0)
	test.Equate(t, tw.Compare(""), true)
}
