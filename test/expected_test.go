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
	"errors"
	"testing"

	"github.com/jetsetilly/gopher2600/test"
)

func TestExpectFailure(t *testing.T) {
	test.ExpectFailure(t, false)
	test.ExpectFailure(t, errors.New("test"))
}

func TestExpectSuccess(t *testing.T) {
	test.ExpectSuccess(t, true)
	var err error
	test.ExpectSuccess(t, err)
	test.ExpectSuccess(t, nil)
}

func TestExpectEquality(t *testing.T) {
	test.ExpectEquality(t, 10, 5+5)
	test.ExpectEquality(t, true, true)
	test.ExpectEquality(t, true, !false)
}

func TestExpectInequality(t *testing.T) {
	test.ExpectInequality(t, 11, 5+5)
	test.ExpectInequality(t, true, false)
}

func TestExpectApproximate(t *testing.T) {
	test.ExpectApproximate(t, 10, 11, 0.1)
}
