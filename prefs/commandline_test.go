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

package prefs_test

import (
	"testing"

	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/test"
)

func TestCommandLineStackValues(t *testing.T) {
	// empty on start
	test.ExpectEquality(t, prefs.PopCommandLineStack(), "")

	// single value
	prefs.PushCommandLineStack("foo::bar")
	test.ExpectEquality(t, prefs.PopCommandLineStack(), "foo::bar")

	// single value but with additional space
	prefs.PushCommandLineStack("   foo:: bar ")
	test.ExpectEquality(t, prefs.PopCommandLineStack(), "foo::bar")

	// more than one key/value in the prefs string. remaining string will
	// will be sorted
	prefs.PushCommandLineStack("foo::bar; baz::qux")
	test.ExpectEquality(t, prefs.PopCommandLineStack(), "baz::qux; foo::bar")

	// check invalid prefs string
	prefs.PushCommandLineStack("foo_bar")
	test.ExpectEquality(t, prefs.PopCommandLineStack(), "")

	// check (partically) invalid prefs string
	prefs.PushCommandLineStack("foo_bar;baz::qux")
	test.ExpectEquality(t, prefs.PopCommandLineStack(), "baz::qux")

	// get prefs value that doesn't exist after pushing a parially invalid prefs string
	prefs.PushCommandLineStack("foo::bar;baz_qux")
	ok, _ := prefs.GetCommandLinePref("baz")
	test.ExpectFailure(t, ok)
	test.ExpectEquality(t, prefs.PopCommandLineStack(), "foo::bar")
}

func TestCommandLineStack(t *testing.T) {
	// empty on start
	test.ExpectEquality(t, prefs.PopCommandLineStack(), "")

	// single value
	prefs.PushCommandLineStack("foo::bar")

	// add another command line group
	prefs.PushCommandLineStack("baz::qux")
	test.ExpectEquality(t, prefs.PopCommandLineStack(), "baz::qux")

	// first group still exists
	test.ExpectEquality(t, prefs.PopCommandLineStack(), "foo::bar")
}
