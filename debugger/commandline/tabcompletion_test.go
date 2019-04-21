package commandline_test

import (
	"gopher2600/debugger/commandline"
	"sort"
	"testing"
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

	// the completer will preserve whitespace
	tc.Reset()
	completion = "FOO   bar     wib"
	expected = "FOO   bar     WIBBLE "
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
		"TEST %V (foo|bar)",
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

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (arg [%V|bar]|foo) %*"})
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
