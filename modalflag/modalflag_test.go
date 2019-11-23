package modalflag_test

import (
	"gopher2600/modalflag"
	"os"
	"testing"
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

type testWriter struct {
	buffer []byte
}

func (tw *testWriter) Write(p []byte) (n int, err error) {
	tw.buffer = append(tw.buffer, p...)
	return len(p), nil
}

func (tw *testWriter) cmp(s string) bool {
	return s == string(tw.buffer)
}

func TestNoHelpAvailable(t *testing.T) {
	tw := &testWriter{}

	md := modalflag.Modes{Output: tw}
	md.NewArgs([]string{"-help"})

	p, _ := md.Parse()
	if p != modalflag.ParseHelp {
		t.Error("expected ParseHelp return value from Parse()")
	}

	if !tw.cmp("No help available\n") {
		t.Error("unexpected help message (wanted 'No help available')")
	}
}

func TestHelpFlags(t *testing.T) {
	tw := &testWriter{}

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

	if !tw.cmp(expectedHelp) {
		t.Error("unexpected help message")
	}
}

func TestHelpModes(t *testing.T) {
	tw := &testWriter{}

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

	if !tw.cmp(expectedHelp) {
		t.Error("unexpected help message")
	}
}

func TestHelpFlagsAndModes(t *testing.T) {
	tw := &testWriter{}

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

	if !tw.cmp(expectedHelp) {
		t.Error("unexpected help message")
	}
}
