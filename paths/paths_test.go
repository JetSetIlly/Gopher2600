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

package paths_test

import (
	"gopher2600/paths"
	"gopher2600/test"
	"testing"
)

func TestPaths(t *testing.T) {
	pth, err := paths.ResourcePath("foo/bar", "baz")
	test.Equate(t, err, nil)
	test.Equate(t, pth, ".gopher2600/foo/bar/baz")

	pth, err = paths.ResourcePath("foo/bar", "")
	test.Equate(t, err, nil)
	test.Equate(t, pth, ".gopher2600/foo/bar")

	pth, err = paths.ResourcePath("", "baz")
	test.Equate(t, err, nil)
	test.Equate(t, pth, ".gopher2600/baz")

	pth, err = paths.ResourcePath("", "")
	test.Equate(t, err, nil)
	test.Equate(t, pth, ".gopher2600")
}
