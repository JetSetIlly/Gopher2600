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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package debugger_test

import (
	"fmt"
	"gopher2600/cartridgeloader"
	"gopher2600/debugger"
	"gopher2600/debugger/terminal"
	"gopher2600/gui"
	"gopher2600/television"
	"io"
	"testing"
	"time"
)

type mockTV struct{}
type mockGUI struct{}

func (t *mockTV) String() string {
	return ""
}

func (t *mockTV) Reset() error {
	return nil
}

func (t *mockTV) AddPixelRenderer(_ television.PixelRenderer) {
}

func (t *mockTV) AddAudioMixer(_ television.AudioMixer) {
}

func (t *mockTV) Signal(_ television.SignalAttributes) error {
	return nil
}

func (t *mockTV) GetState(_ television.StateReq) (int, error) {
	return 0, nil
}

func (t *mockTV) SetSpec(_ string) error {
	return nil
}

func (t *mockTV) GetSpec() *television.Specification {
	return television.SpecNTSC
}

func (t *mockTV) IsStable() bool {
	return true
}

func (t *mockTV) End() error {
	return nil
}

func (g *mockGUI) SetMetaPixel(_ gui.MetaPixel) error {
	return nil
}

func (g *mockGUI) Destroy(_ io.Writer) {
}

func (g *mockGUI) IsVisible() bool {
	return false
}

func (g *mockGUI) SetFeature(request gui.FeatureReq, args ...interface{}) error {
	return nil
}

func (g *mockGUI) SetEventChannel(_ chan (gui.Event)) {
}

func (g *mockGUI) Service() {
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

func (trm *mockTerm) TermRead(buffer []byte, prompt terminal.Prompt, eventChannel chan gui.Event, eventHandler func(gui.Event) error) (int, error) {
	s := <-trm.inp
	copy(buffer, []byte(s))
	return len(s) + 1, nil
}

func (trm *mockTerm) IsInteractive() bool {
	return false
}

func (trm *mockTerm) TermPrintLine(sty terminal.Style, s string, a ...interface{}) {
	trm.out <- fmt.Sprintf(s, a...)
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

func (trm *mockTerm) prtOutput() {
	trm.rcvOutput()
	for i := range trm.output {
		fmt.Println(trm.output[i])
	}
}

// cmpOutput compares the string argument with the *last line* of the most
// recent output. it can easily be adapted to compare the whole output if
// necessary.
func (trm *mockTerm) cmpOutput(s string) bool {
	trm.rcvOutput()

	if len(trm.output) == 0 {
		if len(s) != 0 {
			trm.t.Errorf(fmt.Sprintf("unexpected debugger output (nothinga) should be (%s)", s))
			return false
		}
		return true
	}

	l := len(trm.output) - 1

	if trm.output[l] == s {
		return true
	}

	trm.t.Errorf(fmt.Sprintf("unexpected debugger output (%s) should be (%s)", trm.output[l], s))
	return false
}

func (trm *mockTerm) testSequence() {
	defer func() { trm.sndInput("QUIT") }()
	trm.testBreakpoints()
	trm.testTraps()
	trm.testWatches()
}

func TestDebugger_withNonExistantInitScript(t *testing.T) {
	trm := newMockTerm(t)

	dbg, err := debugger.NewDebugger(&mockTV{}, &mockGUI{}, trm)
	if err != nil {
		t.Fatalf(err.Error())
	}

	go trm.testSequence()

	err = dbg.Start("non_existant_script", cartridgeloader.Loader{})
	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestDebugger(t *testing.T) {
	trm := newMockTerm(t)

	dbg, err := debugger.NewDebugger(&mockTV{}, &mockGUI{}, trm)
	if err != nil {
		t.Fatalf(err.Error())
	}

	go trm.testSequence()

	err = dbg.Start("", cartridgeloader.Loader{})
	if err != nil {
		t.Fatalf(err.Error())
	}
}
