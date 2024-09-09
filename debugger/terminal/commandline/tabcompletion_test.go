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

func TestTabCompletion(t *testing.T) {
	var cmds *commandline.Commands
	var tc *commandline.TabCompletion
	var completion, expected string
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{
		"TEST [arg]",
		"TEST1 [arg]",
		"FOO [bar|baz] wibble",
	})
	test.DemandSuccess(t, err)

	tc = commandline.NewTabCompletion(cmds)

	completion = "TE"
	expected = "TEST "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	// next completion option
	expected = "TEST1 "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	// cycle back to the first completion option
	expected = "TEST "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	tc.Reset()
	completion = "TEST a"
	expected = "TEST ARG "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	tc.Reset()
	completion = "FOO ba"
	expected = "FOO BAR "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	expected = "FOO BAZ "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	// the completion will preserve whitespace
	tc.Reset()
	completion = "FOO   bar     wib"
	expected = "FOO bar WIBBLE "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)
}

func TestTabCompletion_placeholders(t *testing.T) {
	var cmds *commandline.Commands
	var tc *commandline.TabCompletion
	var completion, expected string
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{
		"TEST %P (foo|bar)",
	})
	test.DemandSuccess(t, err)

	tc = commandline.NewTabCompletion(cmds)

	completion = "TEST 100 f"
	expected = "TEST 100 FOO "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)
}

func TestTabCompletion_doubleArgs(t *testing.T) {
	var cmds *commandline.Commands
	var tc *commandline.TabCompletion
	var completion, expected string
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (egg|fog|nug nog|big) (tug)"})
	test.DemandSuccess(t, err)

	tc = commandline.NewTabCompletion(cmds)

	completion = "TEST eg"
	expected = "TEST EGG "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	completion = "TEST egg T"
	expected = "TEST egg TUG "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	completion = "TEST n"
	expected = "TEST NUG "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	completion = "TEST nug N"
	expected = "TEST nug NOG "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	completion = "TEST T"
	expected = "TEST TUG "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	completion = "TEST nug nog T"
	expected = "TEST nug nog TUG "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)
}

func TestTabCompletion_complex(t *testing.T) {
	var cmds *commandline.Commands
	var tc *commandline.TabCompletion
	var completion, expected string
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (arg [%P|bar]|foo)"})
	test.DemandSuccess(t, err)

	tc = commandline.NewTabCompletion(cmds)

	completion = "TEST ar"
	expected = "TEST ARG "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	completion = "TEST arg b"
	expected = "TEST arg BAR "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	completion = "TEST arg 10 wib"
	expected = "TEST arg 10 wib"
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)
}

func TestTabCompletion_filenameFirstOption(t *testing.T) {
	var cmds *commandline.Commands
	var tc *commandline.TabCompletion
	var completion, expected string
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{
		"TEST [%F|foo|bar]",
	})
	test.DemandSuccess(t, err)

	tc = commandline.NewTabCompletion(cmds)

	completion = "TEST f"
	expected = "TEST FOO "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)
}

func TestTabCompletion_nestedGroups(t *testing.T) {
	var cmds *commandline.Commands
	var tc *commandline.TabCompletion
	var completion, expected string
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{
		"TEST [(foo)|bar]",
	})
	test.DemandSuccess(t, err)

	tc = commandline.NewTabCompletion(cmds)

	completion = "TEST f"
	expected = "TEST FOO "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	completion = "TEST FOO bA"
	expected = "TEST FOO bA"
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	completion = "TEST bA"
	expected = "TEST BAR "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	cmds, err = commandline.ParseCommandTemplate([]string{
		"PREF ([SET|NO|TOGGLE] [RANDSTART|RANDPINS])",
	})
	if err != nil {
		t.Errorf("does not parse: %s", err)
	}

	tc = commandline.NewTabCompletion(cmds)
	completion = "P"
	expected = "PREF "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	completion = "PREF S"
	expected = "PREF SET "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	completion = "PREF Tog"
	expected = "PREF TOGGLE "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	completion = "PREF SET R"
	expected = "PREF SET RANDSTART "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	// tab again without changing input
	expected = "PREF SET RANDPINS "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)
}

func TestTabCompletion_repeatGroups(t *testing.T) {
	var cmds *commandline.Commands
	var tc *commandline.TabCompletion
	var completion, expected string
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST {foo}"})
	test.DemandSuccess(t, err)

	tc = commandline.NewTabCompletion(cmds)

	completion = "TEST f"
	expected = "TEST FOO "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	completion = "TEST FOO fo"
	expected = "TEST FOO FOO "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST {foo|bar}"})
	test.DemandSuccess(t, err)

	tc = commandline.NewTabCompletion(cmds)

	completion = "TEST f"
	expected = "TEST FOO "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	completion = "TEST FOO fo"
	expected = "TEST FOO FOO "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)

	completion = "TEST FOO b"
	expected = "TEST FOO BAR "
	completion = tc.Complete(completion)
	test.ExpectEquality(t, completion, expected)
}
