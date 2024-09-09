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
	"strings"
	"testing"

	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"github.com/jetsetilly/gopher2600/test"
)

// wrapper for ExepectEquality in the test package
func expectEquality(t *testing.T, template []string, cmds *commandline.Commands) {
	t.Helper()
	s := strings.Join(template, "\n")
	s = strings.ToUpper(s)
	test.ExpectEquality(t, s, cmds.String())
}

// create a new Commands instance using the output of the supplied Commands. if
// the new Commands output matches the output of the supplied Command instance
// then the test is successful
//
// this is useful for testing that Commands produces correct output that can be
// reused in another context
func expectEquivalency(t *testing.T, cmds *commandline.Commands) {
	t.Helper()

	newCmds, err := commandline.ParseCommandTemplate(strings.Split(cmds.String(), "\n"))
	if test.ExpectSuccess(t, err) {
		test.ExpectEquality(t, cmds.String(), newCmds.String())
	}
}

func TestParser_optimised(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST [1 [2] [3] [4] [5]]"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquivalency(t, cmds)
	}

	template = []string{"TEST (egg|fog|(nug nog)|big) (tug)"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquivalency(t, cmds)
	}
}

func TestParser_nestedGroups(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST (foo|bar (a|b c|d) baz)"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}
}

func TestParser_badGroupings(t *testing.T) {
	var err error

	// optional groups must be closed
	_, err = commandline.ParseCommandTemplate([]string{"TEST (arg"})
	test.ExpectFailure(t, err)

	// required groups must be closed
	_, err = commandline.ParseCommandTemplate([]string{"TEST (arg]"})
	test.ExpectFailure(t, err)
}

func TestParser_goodGroupings(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST (1 [2] [3] [4] [5])"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}
}

func TestParser_nestedGroupings(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST [(foo)|bar]"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST (foo|[bar])"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST (foo|[bar|(baz|qux)]|wibble)"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}
}

func TestParser_rootGroupings(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST (arg)"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}
}

func TestParser_placeholders(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	// placeholder directives must be complete
	_, err = commandline.ParseCommandTemplate([]string{"TEST foo %"})
	test.ExpectFailure(t, err)

	// placeholder directives must be recognised
	_, err = commandline.ParseCommandTemplate([]string{"TEST foo %q"})
	test.ExpectFailure(t, err)

	// double %% is a valid placeholder directive
	template = []string{"TEST foo %%"}
	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	// placeholder directives must be separated from surrounding text
	_, err = commandline.ParseCommandTemplate([]string{"TEST foo%%"})
	test.ExpectFailure(t, err)
}

func TestParser_doubleArgs(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST foo bar"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST (foo bar baz)"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST (egg|fog|nug nog|big) (tug)"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}
}

func TestParser_repeatGroups(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST {foo}"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST {foo|bar}"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST {[foo|bar]}"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST {foo|bar|baz}"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST {foo %f}"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST {foo|bar %f}"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}
}

func TestParser_placeholderLabels(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{
		"FOO %<BAR>S",
	}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectSuccess(t, err) {
		expectEquivalency(t, cmds)
		expectEquality(t, template, cmds)
	}
}

func TestParser_optional(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{
		"FOO (BAR ([BAZ|QUX] [A|B]))",
	}

	cmds, err = commandline.ParseCommandTemplate(template)
	if err != nil {
		t.Errorf("does not parse: %s", err)
	}

	if test.ExpectSuccess(t, err) {
		expectEquivalency(t, cmds)
		expectEquality(t, template, cmds)
	}
}
