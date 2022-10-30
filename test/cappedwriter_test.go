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

package test_test

import (
	"testing"

	"github.com/jetsetilly/gopher2600/test"
)

func TestCappedWriter(t *testing.T) {
	c, err := test.NewCappedWriter(10)
	test.Equate(t, err, nil)

	// testing that the ring writer starts off with the empty string
	test.Equate(t, c.String(), "")

	// add one character
	c.Write([]byte("a"))
	test.Equate(t, c.String(), "a")

	// add another three characters
	c.Write([]byte("bcd"))
	test.Equate(t, c.String(), "abcd")

	// add another six characters, taken us to the limit of 10
	c.Write([]byte("efghij"))
	test.Equate(t, c.String(), "abcdefghij")

	// add another three, which should just be ignoed
	c.Write([]byte("klm"))
	test.Equate(t, c.String(), "abcdefghij")

	// reset and test for empty string
	c.Reset()
	test.Equate(t, c.String(), "")

	// add entire limit in one go
	c.Write([]byte("abcdefghij"))
	test.Equate(t, c.String(), "abcdefghij")

	// reset again
	c.Reset()
	test.Equate(t, c.String(), "")

	// add entire limit and more in one go
	c.Write([]byte("abcdefghijklm"))
	test.Equate(t, c.String(), "abcdefghij")
}
