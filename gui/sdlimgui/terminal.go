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

package sdlimgui

import (
	"gopher2600/debugger/terminal"
	"gopher2600/errors"
	"strings"
)

type term struct {
	inputChan  chan string
	sideChan   chan string
	promptChan chan string
	outputChan chan terminalOutput

	silenced bool

	// when sideChannelSilence is set to true output will not be recorded until
	// output of style Input or Error is received. this system is based on the
	// principal that every command sent by the sideChannel will result in an
	// echo of the input
	sideChannelSilence bool

	tabCompletion terminal.TabCompletion
}

func newTerm() *term {
	return &term{
		// inputChan queue must not block
		inputChan: make(chan string, 1),

		// side-channel terminal input from other areas of the GUI. for
		// example, we can have a menu item that writes "QUIT" to the side
		// channel, with predictable results.
		sideChan: make(chan string, 1),

		promptChan: make(chan string, 1),

		// enormously generous buffer for output channel
		outputChan: make(chan terminalOutput, 1024),
	}
}

// Initialise implements the terminal.Terminal interface
func (trm *term) Initialise() error {
	return nil
}

// CleanUp implements the terminal.Terminal interface
func (trm *term) CleanUp() {
}

// RegisterTabCompletion implements the terminal.Terminal interface
func (trm *term) RegisterTabCompletion(tc terminal.TabCompletion) {
	trm.tabCompletion = tc
}

// Silence implements the terminal.Terminal interface
func (trm *term) Silence(silenced bool) {
	trm.silenced = silenced
}

// TermPrintLine implements the terminal.Output interface
func (trm *term) TermPrintLine(style terminal.Style, s string) {
	if trm.sideChannelSilence && (style == terminal.StyleInput || style == terminal.StyleError) {
		trm.sideChannelSilence = false
		return
	}

	if trm.silenced && style != terminal.StyleError {
		return
	}

	trm.outputChan <- terminalOutput{style: style, text: s}
}

// TermRead implements the terminal.Input interface
func (trm *term) TermRead(buffer []byte, prompt terminal.Prompt, events *terminal.ReadEvents) (int, error) {
	trm.promptChan <- strings.TrimSpace(prompt.Content)

	// the debugger is waiting for input from the terminal but we still need to
	// service gui events in the meantime.
	for {
		select {
		case inp := <-trm.inputChan:
			n := len(inp)
			copy(buffer, inp+"\n")
			return n + 1, nil

		case s := <-trm.sideChan:
			trm.sideChannelSilence = true
			s = strings.TrimSpace(s)
			n := len(s)
			copy(buffer, s+"\n")
			return n + 1, nil

		case ev := <-events.GuiEvents:
			err := events.GuiEventHandler(ev)
			if err != nil {
				return 0, nil
			}

		case _ = <-events.IntEvents:
			return 0, errors.New(errors.UserQuit)
		}
	}
}

// TermRead implements the terminal.Input interface
func (trm *term) TermReadCheck() bool {
	// report on the number of pending items in inputChan and sideChan. if
	// either of these have events waiting then that counts as true
	return len(trm.inputChan) > 0 || len(trm.sideChan) > 0
}

// IsInteractive implements the terminal.Input interface
func (trm *term) IsInteractive() bool {
	return true
}
