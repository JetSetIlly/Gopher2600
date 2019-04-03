package commandline_test

import (
	"gopher2600/debugger/commandline"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/bradleyjkemp/memviz"
)

func expectFailure(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("expected failure")
	}
}

func expectSuccess(t *testing.T, err error) bool {
	t.Helper()
	if err != nil {
		t.Errorf("%s", err)
		return false
	}

	return true
}

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
	if strings.ToUpper(strings.Join(template, "\n")) != cmds.String() {
		t.Errorf("parsed commands do not match template")
		return false
	}
	return true
}

// memvizOutput produces a dot file that can be used to identify how elements
// in the Commands object are connected
func memvizOutput(t *testing.T, filename string, cmds *commandline.Commands) {
	if len(filename) < 4 || strings.ToLower(filename[len(filename)-4:]) != ".dot" {
		filename += ".dot"
	}

	f, err := os.Create(filename)
	defer f.Close()
	if err != nil {
		t.Errorf("%s", err)
	} else {
		memviz.Map(f, cmds)
	}
}

func TestParser_badGroupings(t *testing.T) {
	var err error

	// optional groups must be closed
	_, err = commandline.ParseCommandTemplate([]string{"TEST (arg"})
	expectFailure(t, err)

	// required groups must be closed
	_, err = commandline.ParseCommandTemplate([]string{"TEST (arg]"})
	expectFailure(t, err)
}

func TestParser_goodGroupings(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST (1 [2] [3] [4] [5])"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if expectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	template = []string{"TEST [1 [2] [3] [4] [5]]"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if expectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}
}

func TestParser_rootGroupings(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST (arg) %*"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if expectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}
}

func TestParser_placeholders(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	// placeholder directives must be complete
	_, err = commandline.ParseCommandTemplate([]string{"TEST foo %"})
	expectFailure(t, err)

	// placeholder directives must be recognised
	_, err = commandline.ParseCommandTemplate([]string{"TEST foo %q"})
	expectFailure(t, err)

	// double %% is a valid placeholder directive
	template = []string{"TEST foo %%"}
	cmds, err = commandline.ParseCommandTemplate(template)
	if expectSuccess(t, err) {
		expectEquality(t, template, cmds)
	}

	// placeholder directives must be separated from surrounding text
	cmds, err = commandline.ParseCommandTemplate([]string{"TEST foo%%"})
	expectFailure(t, err)
}

func TestParser_doubleArgs(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{"TEST (egg|fog|nug nog|big) (tug)"}

	cmds, err = commandline.ParseCommandTemplate(template)
	if expectSuccess(t, err) {
		sort.Stable(cmds)
		expectEquality(t, template, cmds)
	}
}

func TestParser(t *testing.T) {
	var template []string
	var cmds *commandline.Commands
	var err error

	template = []string{
		"DISPLAY (OFF|DEBUG|SCALE [%V]|DEBUGCOLORS)",
		"DROP [BREAK|TRAP|WATCH] [%S]",
		"GREP %V",
		"TEST [FOO [%S]|BAR] (EGG [%S]|FOG|NOG NUG) (TUG)",
	}

	cmds, err = commandline.ParseCommandTemplate(template)
	if expectSuccess(t, err) {
		expectEquality(t, template, cmds)
		//memvizOutput(t, "1", cmds)
	}
}
