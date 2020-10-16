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

package sdlimgui

import (
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
)

type term struct {
	// input from the terminal window
	inputChan chan string

	// input from other gui elements (eg. the run button in the control window)
	// only one command can be serviced at a time, which may be inconvenient
	// !!TODO: allow sideChan commands to be queued
	sideChan chan string

	// last string sent to sideChan. we use this to suppress echoing of GUI
	// commands
	lastSideChan string

	// output to the terminal window to present as a prompt
	promptChan chan terminal.Prompt

	// output to the terminal window to present in the main output window
	outputChan chan terminalOutput

	// the state of the last call to Silence()
	silenced bool

	// reference to tab completion. used by terminal window
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

		promptChan: make(chan terminal.Prompt, 1),

		// generous buffer for output channel
		outputChan: make(chan terminalOutput, 4096),
	}
}

// Initialise implements the terminal.Terminal interface.
func (trm *term) Initialise() error {
	return nil
}

// CleanUp implements the terminal.Terminal interface.
func (trm *term) CleanUp() {
}

// RegisterTabCompletion implements the terminal.Terminal interface.
func (trm *term) RegisterTabCompletion(tc terminal.TabCompletion) {
	trm.tabCompletion = tc
}

// Silence implements the terminal.Terminal interface.
func (trm *term) Silence(silenced bool) {
	trm.silenced = silenced
}

// TermPrintLine implements the terminal.Output interface.
func (trm *term) TermPrintLine(style terminal.Style, s string) {
	if trm.silenced && style != terminal.StyleError {
		return
	}

	// do not strings of input style if it is the same as the last string sent
	// to the sideChan
	//
	// this will not suppress echoing of sideChan messages that are not suitably
	// normalised, but that's okay because it will serve as a visual indicator
	// that the sideChan command is not ideal.
	if style == terminal.StyleEcho && s == trm.lastSideChan {
		trm.lastSideChan = ""
		return
	}

	trm.outputChan <- terminalOutput{style: style, text: s}
}

// TermRead implements the terminal.Input interface.
func (trm *term) TermRead(buffer []byte, prompt terminal.Prompt, events *terminal.ReadEvents) (int, error) {
	trm.promptChan <- prompt

	// the debugger is waiting for input from the terminal but we still need to
	// service gui events in the meantime.
	for {
		select {
		case inp := <-trm.inputChan:
			n := len(inp)
			copy(buffer, inp+"\n")
			return n + 1, nil

		case s := <-trm.sideChan:
			s = strings.TrimSpace(s)
			n := len(s)
			copy(buffer, s+"\n")
			trm.lastSideChan = s
			return n + 1, nil

		case ev := <-events.GuiEvents:
			err := events.GuiEventHandler(ev)
			if err != nil {
				return 0, nil
			}

		case ev := <-events.RawEvents:
			ev()

		case <-events.IntEvents:
			return 0, curated.Errorf(terminal.UserAbort)
		}
	}
}

// TermRead implements the terminal.Input interface.
func (trm *term) TermReadCheck() bool {
	// report on the number of pending items in inputChan and sideChan. if
	// either of these have events waiting then that counts as true
	return len(trm.inputChan) > 0 || len(trm.sideChan) > 0
}

// IsInteractive implements the terminal.Input interface.
func (trm *term) IsInteractive() bool {
	return true
}

// where possible the debugger issues commands via the terminal. this has the
// advntage of (a) simplicity and (b) consistency. A QUIT command, for example,
// will work in exactly the same way from the main manu or from the terminal.
//
// to achieve this functionality, the terminal has a side-channel to which a
// complete string is pushed (without a newline character please). the
// pushCommand() is a conveniently placed function to do this.
func (trm *term) pushCommand(input string) {
	// there shouldn't be a problem with channel blocking even though we're
	// issuing and consuming on the same thread. if there is however, we can
	// wrap this channel write in a go call
	trm.sideChan <- input
}
