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

const outputMaxSize = 256

type term struct {
	img *SdlImgui

	tabCompletion terminal.TabCompletion
	history       []string
	historyIdx    int

	silenced bool
	prompt   string
	input    string
	output   []line

	// moreOutput is after TermPrintLine() is executed
	moreOutput bool

	inputEvent chan bool
	sideEvent  chan string
}

func newTerm(img *SdlImgui) (*term, error) {
	term := &term{
		img:        img,
		historyIdx: -1,

		// output is made up of an array of line types. the line type stores
		// the text of the line and the style
		output: make([]line, 0, outputMaxSize),

		// inputEvent queue must not block
		inputEvent: make(chan bool, 1),

		// stuff events can be used for side channel input from other areas
		// of the GUI
		sideEvent: make(chan string, 1),
	}

	term.draw()

	return term, nil
}

type line struct {
	style terminal.Style
	text  string
}

func (l line) draw() {
	switch l.style {
	case terminal.StyleNormalisedInput:
		// white
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{0.8, 0.8, 0.8, 1.0})

	case terminal.StyleHelp:
		// white
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{1.0, 1.0, 1.0, 1.0})

	case terminal.StylePromptCPUStep:
		// white
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{1.0, 1.0, 1.0, 1.0})

	case terminal.StylePromptVideoStep:
		// white
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{0.8, 0.8, 0.8, 1.0})

	case terminal.StylePromptConfirm:
		// blue
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{0.1, 0.4, 0.9, 1.0})

	case terminal.StyleFeedback:
		// white
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{1.0, 1.0, 1.0, 1.0})

	case terminal.StyleCPUStep:
		// yellow
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{0.9, 0.9, 0.5, 1.0})

	case terminal.StyleVideoStep:
		// dimmer yellow
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{0.7, 0.7, 0.3, 1.0})

	case terminal.StyleInstrument:
		// cyan
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{0.1, 0.95, 0.9, 1.0})

	case terminal.StyleFeedbackNonInteractive:
		// white
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{1.0, 1.0, 1.0, 1.0})

	case terminal.StyleError:
		// red
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{0.8, 0.3, 0.3, 1.0})
	}

	imgui.Text(l.text)
	imgui.PopStyleColor()
}

// draw is called by service loop
func (term *term) draw() {
	if term.img.vcs != nil {
		imgui.SetNextWindowPosV(imgui.Vec2{651, 264}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
		imgui.SetNextWindowSizeV(imgui.Vec2{534, 313}, imgui.ConditionFirstUseEver)

		imgui.PushStyleColor(imgui.StyleColorWindowBg, imgui.Vec4{0.1, 0.1, 0.2, 0.8})
		imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{2, 2})
		imgui.BeginV(termTitle, nil, 0)
		imgui.PopStyleVar()
		imgui.PopStyleColor()

		// output
		for i := range term.output {
			term.output[i].draw()
		}
		imgui.Separator()

		// prompt
		imgui.Text(term.prompt)
		imgui.SameLine()

		// this construct says focus the next InputText() box if
		//  - the terminal window is focused
		//  - AND if nothing else has been activated since last frame
		if imgui.IsWindowFocused() && !imgui.IsAnyItemActive() {
			imgui.SetKeyboardFocusHere()
		}

		if imgui.InputTextV("", &term.input,
			imgui.InputTextFlagsEnterReturnsTrue|imgui.InputTextFlagsCallbackCompletion|imgui.InputTextFlagsCallbackHistory,
			term.tabCompleteAndHistory) {
			term.inputEvent <- true
		}

		// add some spacing so that when we scroll to the bottom of the windw
		// it doesn't look goofy
		imgui.Spacing()

		// if output has been added to, scroll to bottom of window
		if term.moreOutput {
			term.moreOutput = false
			imgui.SetScrollHereY(1.0)
		}

		imgui.End()
	}
}

func (term *term) tabCompleteAndHistory(d imgui.InputTextCallbackData) int32 {
	switch d.EventKey() {
	case imgui.KeyTab:
		// tab completion
		b := string(d.Buffer())
		s := term.tabCompletion.Complete(b)
		d.DeleteBytes(0, len(b))
		d.InsertBytes(0, []byte(s))
		d.MarkBufferModified()
	case imgui.KeyUpArrow:
		// previous history item
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
		// next history item
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

// TermPrintLine implements the terminal.Output interface
func (term *term) TermPrintLine(style terminal.Style, s string) {
	if term.silenced && style != terminal.StyleError {
		return
	}

	if len(term.output) >= outputMaxSize {
		term.output = append(term.output[1:], line{style: style, text: s})
	} else {
		term.output = append(term.output, line{style: style, text: s})
	}

	term.moreOutput = true
}

// TermRead implements the terminal.Input interface
func (term *term) TermRead(buffer []byte, prompt terminal.Prompt, events *terminal.ReadEvents) (int, error) {
	term.prompt = prompt.Content

	// the debugger is waiting for input from the terminal but we still need to
	// service gui events in the meantime.
	for {
		select {
		case <-term.inputEvent:
			term.input = strings.TrimSpace(term.input)
			if term.input != "" {
				term.history = append(term.history, term.input)
				term.historyIdx = len(term.history) - 1
			}

			// even if term.input is the empty string we still copy it to the
			// input buffer (sending it back to the caller) because the empty
			// string might mean something

			n := len(term.input)
			copy(buffer, term.input+"\n")
			term.input = ""
			return n + 1, nil

		case s := <-term.sideEvent:
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

// stuff input into the side-channel
func (term *term) stuff(input string) {
	term.sideEvent <- input
}

// TermRead implements the terminal.Input interface
func (term *term) TermReadCheck() bool {
	return len(term.inputEvent) > 0
}

// IsInteractive implements the terminal.Input interface
func (term *term) IsInteractive() bool {
	return true
}
