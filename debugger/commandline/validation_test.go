package commandline_test

import (
	"fmt"
	"gopher2600/debugger/commandline"
	"testing"
)

func TestValidation_required(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST [arg]"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	err = cmds.Validate("TEST arg foo")
	if err == nil {
		t.Errorf("matches but shouldn't")
	} else {
		fmt.Println(err)
	}

	err = cmds.Validate("TEST arg")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}

	err = cmds.Validate("TEST")
	if err == nil {
		t.Errorf("matches but shouldn't")
	} else {
		fmt.Println(err)
	}
}

func TestValidation_optional(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (arg)"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	err = cmds.Validate("TEST")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}

	err = cmds.Validate("TEST arg")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}

	err = cmds.Validate("TEST arg foo")
	if err == nil {
		t.Errorf("matches but shouldn't")
	} else {
		fmt.Println(err)
	}

	err = cmds.Validate("TEST foo")
	if err == nil {
		t.Errorf("matches but shouldn't")
	} else {
		fmt.Println(err)
	}
}

func TestValidation_branchesAndNumeric(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (arg [%V]|foo) %*"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	err = cmds.Validate("TEST")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}

	err = cmds.Validate("TEST foo wibble")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}

	err = cmds.Validate("TEST arg")
	if err == nil {
		t.Errorf("matches but shouldn't")
	} else {
		fmt.Println(err)
	}

	// numeric argument matching
	err = cmds.Validate("TEST arg 10")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}

	// failing a numeric argument match
	err = cmds.Validate("TEST arg bar")
	if err == nil {
		t.Errorf("matches but shouldn't")
	} else {
		fmt.Println(err)
	}

	// ---------------

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (arg|foo) %V"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	err = cmds.Validate("TEST arg")
	if err == nil {
		t.Errorf("matches but shouldn't")
	} else {
		fmt.Println(err)
	}

	err = cmds.Validate("TEST arg 10")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}

	err = cmds.Validate("TEST 10")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}
}

func TestValidation_deepBranches(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	// retry numeric argument matching but with an option for a specific string
	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (arg [%V|bar]|foo) %*"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	err = cmds.Validate("TEST arg bar")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}

	err = cmds.Validate("TEST arg foo")
	if err == nil {
		t.Errorf("matches but shouldn't")
	} else {
		fmt.Println(err)
	}
}

func TestValidation_tripleBranches(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (arg|foo|bar) wibble"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	err = cmds.Validate("TEST foo wibble")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}

	err = cmds.Validate("TEST bar wibble")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}

	err = cmds.Validate("TEST wibble")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}
}

func TestValidation_doubleArgs(t *testing.T) {
	var cmds *commandline.Commands
	var err error

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (nug nog|egg|cream) (tug)"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	err = cmds.Validate("TEST nug nog")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}

	err = cmds.Validate("TEST egg tug")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}

	err = cmds.Validate("TEST nug nog tug")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}

	// ---------------

	cmds, err = commandline.ParseCommandTemplate([]string{"TEST (egg|fog|nug nog|big) (tug)"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	err = cmds.Validate("TEST nug nog")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}

	err = cmds.Validate("TEST fog tug")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}

	err = cmds.Validate("TEST nug nog tug")
	if err != nil {
		t.Errorf("doesn't match but should: %s", err)
	}
}
