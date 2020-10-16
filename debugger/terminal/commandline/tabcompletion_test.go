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
	"sort"
	"testing"

	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
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
	if err != nil {
		t.Fatalf("%s", err)
	}
	sort.Stable(cmds)

	tc = commandline.NewTabCompletion(cmds)

	completion = "TE"
	expected = "TEST "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	// next completion option
	expected = "TEST1 "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	// cycle back to the first completion option
	expected = "TEST "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	tc.Reset()
	completion = "TEST a"
	expected = "TEST ARG "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	tc.Reset()
	completion = "FOO ba"
	expected = "FOO BAR "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	expected = "FOO BAZ "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	// the completion will preserve whitespace
	tc.Reset()
	completion = "FOO   bar     wib"
	expected = "FOO bar WIBBLE "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}
}

func TestTabCompletion_placeholders(t *testing.T) {
	var cmds *commandline.Commands
	var tc *commandline.TabCompletion
	var completion, expected string
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{
		"TEST %P (foo|bar)",
	})
	if err != nil {
		t.Fatalf("%s", err)
	}
	sort.Stable(cmds)

	tc = commandline.NewTabCompletion(cmds)

	completion = "TEST 100 f"
	expected = "TEST 100 FOO "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}
}

func TestTabCompletion_doubleArgs(t *testing.T) {
	var cmds *commandline.Commands
	var tc *commandline.TabCompletion
	var completion, expected string
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (egg|fog|nug nog|big) (tug)"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	tc = commandline.NewTabCompletion(cmds)

	completion = "TEST eg"
	expected = "TEST EGG "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	completion = "TEST egg T"
	expected = "TEST egg TUG "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	completion = "TEST n"
	expected = "TEST NUG "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	completion = "TEST nug N"
	expected = "TEST nug NOG "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	completion = "TEST T"
	expected = "TEST TUG "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	completion = "TEST nug nog T"
	expected = "TEST nug nog TUG "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}
}

func TestTabCompletion_complex(t *testing.T) {
	var cmds *commandline.Commands
	var tc *commandline.TabCompletion
	var completion, expected string
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (arg [%P|bar]|foo)"})
	if err != nil {
		t.Fatalf("%s", err)
	}
	tc = commandline.NewTabCompletion(cmds)

	completion = "TEST ar"
	expected = "TEST ARG "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	completion = "TEST arg b"
	expected = "TEST arg BAR "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	completion = "TEST arg 10 wib"
	expected = "TEST arg 10 wib"
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}
}

func TestTabCompletion_filenameFirstOption(t *testing.T) {
	var cmds *commandline.Commands
	var tc *commandline.TabCompletion
	var completion, expected string
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{
		"TEST [%F|foo|bar]",
	})
	if err != nil {
		t.Fatalf("%s", err)
	}

	tc = commandline.NewTabCompletion(cmds)

	completion = "TEST f"
	expected = "TEST FOO "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}
}

func TestTabCompletion_nestedGroups(t *testing.T) {
	var cmds *commandline.Commands
	var tc *commandline.TabCompletion
	var completion, expected string
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{
		"TEST [(foo)|bar]",
	})
	if err != nil {
		t.Fatalf("%s", err)
	}

	tc = commandline.NewTabCompletion(cmds)

	completion = "TEST f"
	expected = "TEST FOO "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	completion = "TEST FOO bA"
	expected = "TEST FOO bA"
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	completion = "TEST bA"
	expected = "TEST BAR "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

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
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	completion = "PREF S"
	expected = "PREF SET "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	completion = "PREF Tog"
	expected = "PREF TOGGLE "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	completion = "PREF SET R"
	expected = "PREF SET RANDSTART "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	// tab again without changing input
	expected = "PREF SET RANDPINS "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}
}

func TestTabCompletion_repeatGroups(t *testing.T) {
	var cmds *commandline.Commands
	var tc *commandline.TabCompletion
	var completion, expected string
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST {foo}"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	tc = commandline.NewTabCompletion(cmds)

	completion = "TEST f"
	expected = "TEST FOO "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	completion = "TEST FOO fo"
	expected = "TEST FOO FOO "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST {foo|bar}"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	tc = commandline.NewTabCompletion(cmds)

	completion = "TEST f"
	expected = "TEST FOO "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	completion = "TEST FOO fo"
	expected = "TEST FOO FOO "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}

	completion = "TEST FOO b"
	expected = "TEST FOO BAR "
	completion = tc.Complete(completion)
	if completion != expected {
		t.Errorf("expecting '%s' got '%s'", expected, completion)
	}
}
