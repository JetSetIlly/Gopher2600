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

func TestRingWriter(t *testing.T) {
	r, err := test.NewRingWriter(10)
	test.Equate(t, err, nil)

	// testing that the ring writer starts off with the empty string
	test.Equate(t, r.String(), "")

	// writing a short string
	r.Write([]byte("abcde"))
	test.Equate(t, r.String(), "abcde")

	// writing another short string
	r.Write([]byte("fgh"))
	test.Equate(t, r.String(), "abcdefgh")

	// writing another short string that takes the total written the same size
	// as the ring writer's buffer
	r.Write([]byte("ij"))
	test.Equate(t, r.String(), "abcdefghij")

	// writing another short string that takes the written string beyond the
	// size of the ring writer's buffer
	r.Write([]byte("kl"))
	test.Equate(t, r.String(), "cdefghijkl")
	r.Write([]byte("mn"))
	test.Equate(t, r.String(), "efghijklmn")

	// writing a string the same length as the ring writer's buffer. when there
	// is already content in the ring writer
	r.Write([]byte("1234567890"))
	test.Equate(t, r.String(), "1234567890")

	// writing a string that is longer than the ring writer's buffer. when
	// there is already content in the ring writer
	r.Write([]byte("1234567890ABC"))
	test.Equate(t, r.String(), "4567890ABC")

	// reseting the buffer and then writing a string that is longer than the
	// ring writer's buffer
	r.Reset()
	test.Equate(t, r.String(), "")
	r.Write([]byte("1234567890ABC"))
	test.Equate(t, r.String(), "4567890ABC")
}
