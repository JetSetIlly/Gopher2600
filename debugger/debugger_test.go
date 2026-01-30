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

package debugger_test

import (
	"testing"
	"time"

	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"github.com/jetsetilly/gopher2600/gui"
)

type mockGUI struct{}

func (g *mockGUI) SetFeature(request gui.FeatureReq, args ...gui.FeatureReqData) error {
	return nil
}

type mockTerm struct {
	t      *testing.T
	inp    chan string
	out    chan string
	output []string
}

func newMockTerm(t *testing.T) *mockTerm {
	trm := &mockTerm{
		t:   t,
		inp: make(chan string),
		out: make(chan string, 100),
	}
	return trm
}

// Initialise implements the terminal.Output interface
func (trm *mockTerm) Initialise() error {
	return nil
}

// CleanUp implements the terminal.Output interface
func (trm *mockTerm) CleanUp() {
}

// RegisterTabCompletion implements the terminal.Output interface
func (trm *mockTerm) RegisterTabCompletion(_ *commandline.TabCompletion) {
}

// Silence implements the terminal.Output interface
func (trm *mockTerm) Silence(silenced bool) {
}

// TermRead implements the terminal.Output interface
func (trm *mockTerm) TermRead(buffer []byte, _ terminal.Prompt, _ *terminal.ReadEvents) (int, error) {
	s := <-trm.inp
	copy(buffer, s)
	return len(s) + 1, nil
}

// TermReadCheck implements the terminal.Output interface
func (trm *mockTerm) TermReadCheck() bool {
	return false
}

// IsInteractive implements the terminal.Output interface
func (trm *mockTerm) IsInteractive() bool {
	return false
}

// IsRealTerminal implements the terminal.Output interface
func (trm *mockTerm) IsRealTerminal() bool {
	return false
}

// TermPrintLine implements the terminal.Output interface
func (trm *mockTerm) TermPrintLine(sty terminal.Style, s string) {
	if sty == terminal.StyleEcho {
		return
	}

	trm.out <- s
}

func (trm *mockTerm) command(s string) {
	trm.output = make([]string, 0, 10)
	trm.inp <- s
}

func (trm *mockTerm) response() {
	empty := false
	for !empty {
		select {
		case s := <-trm.out:
			trm.output = append(trm.output, s)

		// the amount of output sent by the debugger is unpredictable so a
		// timeout is necessary. a matter of milliseconds should be sufficient
		case <-time.After(10 * time.Millisecond):
			empty = true
		}
	}
}

func (trm *mockTerm) lastLine() string {
	trm.response()
	if len(trm.output) == 0 {
		return ""
	}
	return trm.output[len(trm.output)-1]
}

func testSequence(t *testing.T, trm *mockTerm) {
	defer func() {
		trm.command("QUIT")
	}()
	testBreakpoints(t, trm)
	testBreakpoints_drop(t, trm)
	testTraps(t, trm)
	testWatches(t, trm)
}

func TestDebugger_withNonExistantInitScript(t *testing.T) {
	var trm *mockTerm

	create := func(dbg *debugger.Debugger) (gui.GUI, terminal.Terminal, error) {
		trm = newMockTerm(t)
		return &mockGUI{}, trm, nil
	}

	var opts debugger.CommandLineOptions

	dbg, err := debugger.NewDebugger(opts, create)
	if err != nil {
		t.Fatal(err.Error())
	}

	// panic on any error from start debugger function
	go func() {
		err := dbg.StartInDebugMode("")
		if err != nil {
			panic(err)
		}
	}()

	testSequence(t, trm)
}

func TestDebugger(t *testing.T) {
	var trm *mockTerm

	create := func(dbg *debugger.Debugger) (gui.GUI, terminal.Terminal, error) {
		trm = newMockTerm(t)
		return &mockGUI{}, trm, nil
	}

	var opts debugger.CommandLineOptions

	dbg, err := debugger.NewDebugger(opts, create)
	if err != nil {
		t.Fatal(err.Error())
	}

	// panic on any error from start debugger function
	go func() {
		err := dbg.StartInDebugMode("")
		if err != nil {
			panic(err)
		}
	}()

	testSequence(t, trm)
}
