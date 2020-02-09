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

	"github.com/inkyblackness/imgui-go/v2"
)

const termTitle = "Terminal"

type term struct {
	img   *SdlImgui
	setup bool

	tabCompletion terminal.TabCompletion
	history       []string
	historyIdx    int

	prompt string
	input  string
	output strings.Builder

	// activateInput is true if the return key was pressed in the terminal input
	// box during the last iteration. the return key makes the input box lose
	// focus so we need to know to activate it again.
	activateInput bool

	silenced bool

	inputEvent chan bool
}

func newTerm(img *SdlImgui) (*term, error) {
	term := &term{
		img:        img,
		inputEvent: make(chan bool),
		historyIdx: -1,
	}

	return term, nil
}

// draw is called by service loop
func (term *term) draw() {
	if term.img.vcs != nil {
		if !term.setup {
			imgui.SetNextWindowPos(imgui.Vec2{651, 264})
			size := imgui.Vec2{534, 313}
			imgui.SetNextWindowSize(size)
			term.setup = true
		}
		imgui.BeginV(termTitle, nil, 0)

		// output
		imgui.Text(term.output.String())

		imgui.Separator()

		// prompt
		if term.activateInput {
			imgui.SetKeyboardFocusHere(-1)
			term.activateInput = false
		}
		imgui.Text(term.prompt)
		imgui.SameLine()
		if imgui.InputTextV("", &term.input,
			imgui.InputTextFlagsEnterReturnsTrue|imgui.InputTextFlagsCallbackCompletion|imgui.InputTextFlagsCallbackHistory,
			term.tabComplete) {
			term.inputEvent <- true
			term.activateInput = true
		}

		imgui.End()
	}
}

func (term *term) tabComplete(d imgui.InputTextCallbackData) int32 {
	switch d.EventKey() {
	case imgui.KeyTab:
		b := string(d.Buffer())
		s := term.tabCompletion.Complete(b)
		d.DeleteBytes(0, len(b))
		d.InsertBytes(0, []byte(s))
		d.MarkBufferModified()
	case imgui.KeyUpArrow:
		if term.historyIdx > -1 {
			b := string(d.Buffer())
			d.DeleteBytes(0, len(b))
			d.InsertBytes(0, []byte(term.history[term.historyIdx]))
			if term.historyIdx > 0 {
				term.historyIdx--
			}
			d.MarkBufferModified()
		}
	case imgui.KeyDownArrow:
		if term.historyIdx < len(term.history)-1 {
			b := string(d.Buffer())
			d.DeleteBytes(0, len(b))
			d.InsertBytes(0, []byte(term.history[term.historyIdx]))
			if term.historyIdx < len(term.history)-1 {
				term.historyIdx++
			}
		} else {
			b := string(d.Buffer())
			d.DeleteBytes(0, len(b))
		}
		d.MarkBufferModified()
	}
	return 0
}

// Initialise implements the terminal.Terminal interface
func (term *term) Initialise() error {
	return nil
}

// CleanUp implements the terminal.Terminal interface
func (term *term) CleanUp() {
}

// RegisterTabCompletion implements the terminal.Terminal interface
func (term *term) RegisterTabCompletion(tc terminal.TabCompletion) {
	term.tabCompletion = tc
}

// Silence implements the terminal.Terminal interface
func (term *term) Silence(silenced bool) {
	term.silenced = silenced
}

// TermPrintLine implements the terminal.Terminal interface
func (term *term) TermPrintLine(style terminal.Style, s string) {
	term.output.WriteString(s)
	term.output.WriteString("\n")
}

// TermRead implements the terminal.Terminal interface
func (term *term) TermRead(buffer []byte, prompt terminal.Prompt, events *terminal.ReadEvents) (int, error) {
	term.prompt = prompt.Content

	// the debugger is waiting for input from the terminal but we still need to
	// service gui events in the meantime.
	for {
		select {
		case <-term.inputEvent:
			term.history = append(term.history, term.input)
			term.historyIdx = len(term.history) - 1

			n := len(term.input)
			copy(buffer, term.input+"\n")
			term.input = ""
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

// IsInteractive implements the terminal.Terminal interface
func (term *term) IsInteractive() bool {
	return true
}
