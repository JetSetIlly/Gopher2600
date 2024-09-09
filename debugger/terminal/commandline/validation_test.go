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

func TestValidation_required(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST [arg]"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST arg foo")
	if test.ExpectFailure(t, err) {
		test.ExpectEquality(t, err.Error(), "unrecognised argument (foo) for TEST")
	}

	err = cmds.Validate("TEST arg")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("TEST")
	if test.ExpectFailure(t, err) {
		test.ExpectEquality(t, err.Error(), "ARG required")
	}
}

func TestValidation_optional(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (arg)"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("TEST arg")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("TEST arg foo")
	test.ExpectFailure(t, err)

	err = cmds.Validate("TEST foo")
	test.ExpectFailure(t, err)
}

func TestValidation_optional2(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (arg [%s]|bar)"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST xxxxx")
	if test.ExpectFailure(t, err) {
		test.ExpectEquality(t, err.Error(), "unrecognised argument (xxxxx) for TEST")
	}
}

func TestValidation_branchesAndNumeric(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (arg [%N]|foo)"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("TEST arg")
	test.ExpectFailure(t, err)

	// numeric argument matching
	err = cmds.Validate("TEST arg 10")
	test.ExpectSuccess(t, err)

	// failing a numeric argument match
	err = cmds.Validate("TEST arg bar")
	test.ExpectFailure(t, err)

	// ---------------

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (arg|foo) %N"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST arg")
	test.ExpectFailure(t, err)

	err = cmds.Validate("TEST arg 10")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("TEST 10")
	test.ExpectSuccess(t, err)
}

func TestValidation_deepBranches(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	// retry numeric argument matching but with an option for a specific string
	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (arg [%N|bar]|foo)"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST arg bar")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("TEST arg foo")
	test.ExpectFailure(t, err)
}

func TestValidation_tripleBranches(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (arg|foo|bar) wibble"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST foo wibble")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("TEST bar wibble")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("TEST wibble")
	test.ExpectSuccess(t, err)
}

func TestValidation_doubleArgs(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (nug nog|egg|cream) (tug)"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST nug nog")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("TEST egg tug")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("TEST nug nog tug")
	test.ExpectSuccess(t, err)

	// ---------------

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (egg|fog|nug nog|big) (tug)"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST nug nog")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("TEST fog tug")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("TEST nug nog tug")
	test.ExpectSuccess(t, err)
}

func TestValidation_filenameFirstArg(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST [%F|foo [wibble]|bar]"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST foo wibble")
	test.ExpectSuccess(t, err)
}

func TestValidation_singluarOption(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"SCRIPT [RECORD (REGRESSION) [%S]|END|%F]"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("SCRIPT foo")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("SCRIPT END")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("SCRIPT RECORD foo")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("SCRIPT RECORD REGRESSION foo")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("SCRIPT RECORD REGRESSION foo end")
	test.ExpectFailure(t, err)
}

func TestValidation_nestedGroups(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST [(foo|baz)|bar]"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST foo")
	test.ExpectSuccess(t, err)
	err = cmds.Validate("TEST bar")
	test.ExpectSuccess(t, err)
	err = cmds.Validate("TEST wibble")
	test.ExpectFailure(t, err)

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (foo|[bar|(baz|qux)]|wibble)"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST foo")
	test.ExpectSuccess(t, err)
	err = cmds.Validate("TEST wibble")
	test.ExpectSuccess(t, err)
	err = cmds.Validate("TEST bar")
	test.ExpectSuccess(t, err)
}

func TestValidation_repeatGroups(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST {foo}"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST foo")
	test.ExpectSuccess(t, err)
	err = cmds.Validate("TEST foo foo")
	test.ExpectSuccess(t, err)

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST {foo|bar|baz}"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST foo")
	test.ExpectSuccess(t, err)
	err = cmds.Validate("TEST foo foo")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("TEST bar foo")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("TEST bar foo baz baz")
	test.ExpectSuccess(t, err)

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST [foo|bar {baz|qux}]"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST foo")
	test.ExpectSuccess(t, err)
	err = cmds.Validate("TEST bar")
	test.ExpectSuccess(t, err)
	err = cmds.Validate("TEST bar baz")
	test.ExpectSuccess(t, err)
	err = cmds.Validate("TEST bar baz qux")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("TEST foo bar")
	test.ExpectFailure(t, err)

	err = cmds.Validate("TEST bar baz bar")
	test.ExpectFailure(t, err)

	err = cmds.Validate("TEST bar baz qux qux baz wibble")
	test.ExpectFailure(t, err)

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST {[foo]}"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST")
	test.ExpectSuccess(t, err)
	err = cmds.Validate("TEST foo")
	test.ExpectSuccess(t, err)
	err = cmds.Validate("TEST foo foo")
	test.ExpectSuccess(t, err)

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST {(foo)}"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("TEST")
	test.ExpectSuccess(t, err)
	err = cmds.Validate("TEST foo")
	test.ExpectSuccess(t, err)
	err = cmds.Validate("TEST foo foo")
	test.ExpectSuccess(t, err)
}

func TestValidation_foo(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"SYMBOL [%S (ALL|MIRRORS)|LIST]"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("SYMBOL enabl")
	test.ExpectSuccess(t, err)
}

func TestValidation_bar(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{
		"LIST",
		"PRINT [%s]",
		"SORT (RISING|FALLING)",
	})
	test.DemandSuccess(t, err)

	err = cmds.Validate("list")
	test.ExpectSuccess(t, err)
}

func TestValidation_optional_group(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{
		"PREF [SET|NO|TOGGLE] [RANDSTART|RANDPINS]",
	})
	test.DemandSuccess(t, err)

	err = cmds.Validate("pref")
	if test.ExpectFailure(t, err) {
		test.ExpectEquality(t, err.Error(), "SET or NO or TOGGLE required")
	}

	err = cmds.Validate("pref set")
	if test.ExpectFailure(t, err) {
		test.ExpectEquality(t, err.Error(), "RANDSTART or RANDPINS required")
	}

	err = cmds.Validate("pref set randstart")
	test.ExpectSuccess(t, err)

	// same as above except that the required argument sequence (in its
	// entirity) is optional

	cmds, err = commandline.ParseCommandTemplate([]string{
		"PREF ([SET|NO|TOGGLE] [RANDSTART|RANDPINS])",
	})
	test.ExpectSuccess(t, err)

	err = cmds.Validate("pref")
	test.ExpectSuccess(t, err)

	err = cmds.Validate("pref set")
	if test.ExpectFailure(t, err) {
		test.ExpectEquality(t, err.Error(), "RANDSTART or RANDPINS required")
	}

	err = cmds.Validate("pref set randstart")
	test.ExpectSuccess(t, err)
}

func TestValidation_BREAK_style(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"YYYYY [%s %n| %s] {& %s %n|& %s}"})
	test.DemandSuccess(t, err)

	err = cmds.Validate("YYYYY SL 100")
	test.ExpectSuccess(t, err)
}
