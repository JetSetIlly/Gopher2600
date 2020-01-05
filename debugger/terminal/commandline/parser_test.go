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

package commandline_test

import (
	"gopher2600/debugger/terminal/commandline"
	"gopher2600/test"
	"strings"
	"testing"
)

// expectEquality compares a template, as passed to ParseCommandTemplate(),
// with the String() output of the resulting Commands object. both outputs
// should be the same.
//
// the template is transformed slightly. each entry in the array is joined with
// a newline character and also converted to uppercase. this is okay because
// we're only really interested in how the groupings and branching is
// represented.
func expectEquality(t *testing.T, template []string, cmds *commandline.Commands) bool {
	t.Helper()
	if strings.ToUpper(strings.Join(template, "\n")) != strings.ToUpper(cmds.String()) {
		t.Errorf("parsed commands do not match template")
		return false
	}
	return true
}

// dur to the parsing method it's not always possible to recreate the original
// template from the parsed nodes. but that's okay, the parsed nodes have
// effectively been optimised. in these test cases, rather than using the
// expectEquality() function, we can use this expectEquivalency() function.
//
// rather than using the original template, this function runs the result of
// the parsed Commands back through itself. if the results of the second pass
// are the same as the first then we've successfully parsed the original
// template.
func expectEquivalency(t *testing.T, cmds *commandline.Commands) bool {
	t.Helper()

	var err error

	template := strings.Split(cmds.String(), "\n")
	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		return expectEquality(t, template, cmds)
	}

	return false
}

func TestParser_optimised(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST [1 [2] [3] [4] [5]]"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquivalency(t, cmds)
	}

	template = []string{"TEST (egg|fog|(nug nog)|big) (tug)"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquivalency(t, cmds)
	}
}

func TestParser_nestedGroups(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST (foo|bar (a|b c|d) baz)"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}
}

func TestParser_badGroupings(t *testing.T) {
	var err error

	// optional groups must be closed
	_, err = commandline.ParseCommandTemplate([]string{"TEST (arg"})
	test.ExpectedFailure(t, err)

	// required groups must be closed
	_, err = commandline.ParseCommandTemplate([]string{"TEST (arg]"})
	test.ExpectedFailure(t, err)
}

func TestParser_goodGroupings(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST (1 [2] [3] [4] [5])"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

}

func TestParser_nestedGroupings(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST [(foo)|bar]"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST (foo|[bar])"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST (foo|[bar|(baz|qux)]|wibble)"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}
}

func TestParser_rootGroupings(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST (arg)"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}
}

func TestParser_placeholders(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	// placeholder directives must be complete
	_, err = commandline.ParseCommandTemplate([]string{"TEST foo %"})
	test.ExpectedFailure(t, err)

	// placeholder directives must be recognised
	_, err = commandline.ParseCommandTemplate([]string{"TEST foo %q"})
	test.ExpectedFailure(t, err)

	// double %% is a valid placeholder directive
	template = []string{"TEST foo %%"}
	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	// placeholder directives must be separated from surrounding text
	_, err = commandline.ParseCommandTemplate([]string{"TEST foo%%"})
	test.ExpectedFailure(t, err)
}

func TestParser_doubleArgs(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST foo bar"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST (foo bar baz)"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST (egg|fog|nug nog|big) (tug)"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}
}

func TestParser_repeatGroups(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST {foo}"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST {foo|bar}"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST {[foo|bar]}"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST {foo|bar|baz}"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST {foo %f}"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST {foo|bar %f}"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}
}

func TestParser_addHelp(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{
		"DISPLAY (OFF|DEBUG|SCALE [%N]|DEBUGCOLORS)",
		"SCRIPT [%F|RECORD %F|END]",
		"DROP [BREAK|TRAP|WATCH] [%S]",
		"GREP %N",
		"SYMBOL [%S (ALL|MIRRORS)|LIST]",
	}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	err = cmds.AddHelp("HELP", map[string]string{})
	test.ExpectedSuccess(t, err)

	// adding a second HELP command is not allowed
	err = cmds.AddHelp("HELP", map[string]string{})
	test.ExpectedFailure(t, err)
}

func TestParser_placeholderLabels(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{
		"FOO %<bar>S",
	}

	cmds, err = commandline.ParseCommandTemplate(template)
	if test.ExpectedSuccess(t, err) {
		expectEquivalency(t, cmds)
		expectEquality(t, template, cmds)
	}
}
