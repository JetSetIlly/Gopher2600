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

	prompt string
	input  string
	output strings.Builder

	silenced bool

	inputEvent chan bool
}

func newTerm(img *SdlImgui) (*term, error) {
	term := &term{
		img:        img,
		inputEvent: make(chan bool),
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
		imgui.Text(term.prompt)
		imgui.SameLine()
		if imgui.InputTextV("", &term.input, imgui.InputTextFlagsEnterReturnsTrue, nil) {
			term.inputEvent <- true
		}

		imgui.End()
	}
}

// Initialise implements the terminal.Terminal interface
func (term *term) Initialise() error {
	return nil
}

// CleanUp implements the terminal.Terminal interface
func (term *term) CleanUp() {
}

// RegisterTabCompletion implements the terminal.Terminal interface
func (term *term) RegisterTabCompletion(terminal.TabCompletion) {
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
