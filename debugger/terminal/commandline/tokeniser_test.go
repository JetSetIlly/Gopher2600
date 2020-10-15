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

package commandline_test

import (
	"testing"

	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"github.com/jetsetilly/gopher2600/test"
)

func TestTokeniser_spaces(t *testing.T) {
	var toks *commandline.Tokens
	var s string

	toks = commandline.TokeniseInput("FOO")
	test.Equate(t, toks.Len(), 1)
	s, _ = toks.Get()
	test.Equate(t, s, "FOO")

	toks = commandline.TokeniseInput("FOO ")
	test.Equate(t, toks.Len(), 1)
	s, _ = toks.Get()
	test.Equate(t, s, "FOO")

	toks = commandline.TokeniseInput("FOO   BAR")
	test.Equate(t, toks.Len(), 2)
	s, _ = toks.Get()
	test.Equate(t, s, "FOO")
	s, _ = toks.Get()
	test.Equate(t, s, "BAR")

	toks = commandline.TokeniseInput("    FOO   BAR  ")
	test.Equate(t, toks.Len(), 2)
	s, _ = toks.Get()
	test.Equate(t, s, "FOO")
	s, _ = toks.Get()
	test.Equate(t, s, "BAR")

	toks = commandline.TokeniseInput("    FOO   BAR  BAZ")
	test.Equate(t, toks.Len(), 3)
	s, _ = toks.Get()
	test.Equate(t, s, "FOO")
	s, _ = toks.Get()
	test.Equate(t, s, "BAR")
	s, _ = toks.Get()
	test.Equate(t, s, "BAZ")
}

func TestTokeniser_quotes(t *testing.T) {
	var toks *commandline.Tokens
	var s string

	// last argument is quoted
	toks = commandline.TokeniseInput("FOO \"BAR  BAZ\"  ")
	test.Equate(t, toks.Len(), 2)
	s, _ = toks.Get()
	test.Equate(t, s, "FOO")
	s, _ = toks.Get()
	test.Equate(t, s, "BAR  BAZ")

	// middle argument is quoted
	toks = commandline.TokeniseInput("FOO \"BAR  BAZ\" QUX")
	test.Equate(t, toks.Len(), 3)
	s, _ = toks.Get()
	test.Equate(t, s, "FOO")
	s, _ = toks.Get()
	test.Equate(t, s, "BAR  BAZ")
	s, _ = toks.Get()
	test.Equate(t, s, "QUX")

	// first argument is quoted
	toks = commandline.TokeniseInput("\"FOO BAR\" BAZ   QUX")
	test.Equate(t, toks.Len(), 3)
	s, _ = toks.Get()
	test.Equate(t, s, "FOO BAR")
	s, _ = toks.Get()
	test.Equate(t, s, "BAZ")
	s, _ = toks.Get()
	test.Equate(t, s, "QUX")

	// the only argument is quoted and with leadig and trailing space
	toks = commandline.TokeniseInput("  \"  FOO BAR    \" ")
	test.Equate(t, toks.Len(), 1)
	s, _ = toks.Get()
	test.Equate(t, s, "  FOO BAR    ")
}

func TestTokeniser_singleCharArgs(t *testing.T) {
	toks := commandline.TokeniseInput("FOO & BAR")
	test.Equate(t, toks.Len(), 3)
}
