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
	"fmt"
	"testing"
	"time"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware/television"
)

type mockGUI struct{}

func (g *mockGUI) ReqFeature(request gui.FeatureReq, args ...interface{}) error {
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

func (trm *mockTerm) Initialise() error {
	return nil
}

func (trm *mockTerm) CleanUp() {
}

func (trm *mockTerm) RegisterTabCompletion(_ terminal.TabCompletion) {
}

func (trm *mockTerm) Silence(silenced bool) {
}

func (trm *mockTerm) TermRead(buffer []byte, _ terminal.Prompt, _ *terminal.ReadEvents) (int, error) {
	s := <-trm.inp
	copy(buffer, s)
	return len(s) + 1, nil
}

func (trm *mockTerm) TermReadCheck() bool {
	return false
}

func (trm *mockTerm) IsInteractive() bool {
	return false
}

func (trm *mockTerm) TermPrintLine(sty terminal.Style, s string) {
	if sty == terminal.StyleEcho {
		return
	}

	trm.out <- s
}

func (trm *mockTerm) sndInput(s string) {
	trm.output = make([]string, 0, 10)
	trm.inp <- s
}

func (trm *mockTerm) rcvOutput() {
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

// cmpOutput compares the string argument with the *last line* of the most
// recent output. it can easily be adapted to compare the whole output if
// necessary.
func (trm *mockTerm) cmpOutput(s string) {
	trm.rcvOutput()

	if len(trm.output) == 0 {
		if len(s) != 0 {
			trm.t.Errorf(fmt.Sprintf("unexpected debugger output (nothing) should be (%s)", s))
			return
		}
		return
	}

	l := len(trm.output) - 1

	if trm.output[l] == s {
		return
	}

	trm.t.Errorf(fmt.Sprintf("unexpected debugger output (%s) should be (%s)", trm.output[l], s))
}

func (trm *mockTerm) testSequence() {
	defer func() { trm.sndInput("QUIT") }()
	trm.testBreakpoints()
	trm.testTraps()
	trm.testWatches()
}

func TestDebugger_withNonExistantInitScript(t *testing.T) {
	trm := newMockTerm(t)

	tv, err := television.NewTelevision("NTSC")
	if err != nil {
		t.Fatalf(err.Error())
	}

	dbg, err := debugger.NewDebugger(tv, &mockGUI{}, trm, false)
	if err != nil {
		t.Fatalf(err.Error())
	}

	go trm.testSequence()

	err = dbg.Start("non_existent_script", cartridgeloader.Loader{})
	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestDebugger(t *testing.T) {
	trm := newMockTerm(t)
	tv, err := television.NewTelevision("NTSC")
	if err != nil {
		t.Fatalf(err.Error())
	}

	dbg, err := debugger.NewDebugger(tv, &mockGUI{}, trm, false)
	if err != nil {
		t.Fatalf(err.Error())
	}

	go trm.testSequence()

	err = dbg.Start("", cartridgeloader.Loader{})
	if err != nil {
		t.Fatalf(err.Error())
	}
}
