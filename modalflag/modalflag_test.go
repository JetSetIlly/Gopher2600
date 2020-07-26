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

package modalflag_test

import (
	"os"
	"testing"

	"github.com/jetsetilly/gopher2600/modalflag"
	"github.com/jetsetilly/gopher2600/test"
)

func TestNoModesNoFlags(t *testing.T) {
	md := modalflag.Modes{Output: os.Stdout}
	md.NewArgs([]string{})

	p, err := md.Parse()
	if p != modalflag.ParseContinue {
		t.Error("expected ParseContinue")
	}
	if err != nil {
		t.Errorf("did not expect error: %s", err)
	}
	if md.Mode() != "" {
		t.Errorf("did not expect to see mode as result of Parse()")
	}
	if md.Path() != "" {
		t.Errorf("did not expect to see modes in mode path")
	}
}

func TestNoModes(t *testing.T) {
	md := modalflag.Modes{Output: os.Stdout}
	md.NewArgs([]string{"-test", "1", "2"})
	testFlag := md.AddBool("test", false, "test flag")

	if *testFlag != false {
		t.Error("expected *testFlag to be false before Parse()")
	}

	p, err := md.Parse()
	if p != modalflag.ParseContinue {
		t.Error("expected ParseContinue")
	}
	if err != nil {
		t.Errorf("did not expect error: %s", err)
	}
	if md.Mode() != "" {
		t.Errorf("did not expect to see mode as result of Parse()")
	}
	if md.Path() != "" {
		t.Errorf("did not expect to see modes in mode path")
	}

	if *testFlag != true {
		t.Error("expected *testFlag to be true after Parse()")
	}

	if len(md.RemainingArgs()) != 2 {
		t.Error("expected number of RemainingArgs() to be 2 after Parse()")
	}
}

func TestNoHelpAvailable(t *testing.T) {
	tw := &test.Writer{}

	md := modalflag.Modes{Output: tw}
	md.NewArgs([]string{"-help"})

	p, _ := md.Parse()
	if p != modalflag.ParseHelp {
		t.Error("expected ParseHelp return value from Parse()")
	}

	if !tw.Compare("No help available\n") {
		t.Error("unexpected help message (wanted 'No help available')")
	}
}

func TestHelpFlags(t *testing.T) {
	tw := &test.Writer{}

	md := modalflag.Modes{Output: tw}
	md.NewArgs([]string{"-help"})
	md.AddBool("test", true, "test flag")

	p, _ := md.Parse()
	if p != modalflag.ParseHelp {
		t.Error("expected ParseHelp return value from Parse()")
	}

	expectedHelp := "Usage:\n" +
		"  -test\n" +
		"    	test flag (default true)\n"

	if !tw.Compare(expectedHelp) {
		t.Error("unexpected help message")
	}
}

func TestHelpModes(t *testing.T) {
	tw := &test.Writer{}

	md := modalflag.Modes{Output: tw}
	md.NewArgs([]string{"-help"})
	md.AddSubModes("A", "B", "C")

	p, _ := md.Parse()
	if p != modalflag.ParseHelp {
		t.Error("expected ParseHelp return value from Parse()")
	}

	expectedHelp := "Usage:\n" +
		"  available sub-modes: A, B, C\n" +
		"    default: A\n"

	if !tw.Compare(expectedHelp) {
		t.Error("unexpected help message")
	}
}

func TestHelpFlagsAndModes(t *testing.T) {
	tw := &test.Writer{}

	md := modalflag.Modes{Output: tw}
	md.NewArgs([]string{"-help"})
	md.AddBool("test", true, "test flag")
	md.AddSubModes("A", "B", "C")

	p, _ := md.Parse()
	if p != modalflag.ParseHelp {
		t.Error("expected ParseHelp return value from Parse()")
	}

	expectedHelp := "Usage:\n" +
		"  -test\n" +
		"    	test flag (default true)\n" +
		"\n" +
		"  available sub-modes: A, B, C\n" +
		"    default: A\n"

	if !tw.Compare(expectedHelp) {
		t.Error("unexpected help message")
	}
}
