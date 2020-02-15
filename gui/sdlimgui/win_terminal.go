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

const (
	outputMaxSize = 256
)

type term struct {
	windowManagement
	img *SdlImgui

	tabCompletion terminal.TabCompletion
	history       []string
	historyIdx    int

	silenced bool
	prompt   string
	input    string
	output   []terminalOutput

	// moreOutput is after TermPrintLine() is executed
	moreOutput bool

	inputChan chan bool
	sideChan  chan string
}

func newTerm(img *SdlImgui) (managedWindow, error) {
	trm := &term{
		img:        img,
		historyIdx: -1,

		// output is made up of an array of line types. the line type stores
		// the text of the line and the style
		output: make([]terminalOutput, 0, outputMaxSize),

		// inputChan queue must not block
		inputChan: make(chan bool, 1),

		// side-channel terminal input from other areas of the GUI. for
		// example, we can have a menu item that writes "QUIT" to the side
		// channel, with predictable results.
		sideChan: make(chan string, 1),
	}

	return trm, nil
}

func (trm *term) destroy() {
}

func (trm *term) id() string {
	return termTitle
}

// draw is called by service loop
func (trm *term) draw() {
	if !trm.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{369, 274}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{534, 313}, imgui.ConditionFirstUseEver)

	imgui.PushStyleColor(imgui.StyleColorWindowBg, imgui.Vec4{0.1, 0.1, 0.2, 0.9})
	imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{2, 2})
	imgui.BeginV(termTitle, &trm.open, 0)
	imgui.PopStyleVar()
	imgui.PopStyleColor()

	// output
	for i := range trm.output {
		trm.output[i].draw()
	}
	imgui.Separator()

	// prompt
	imgui.Text(trm.prompt)
	imgui.SameLine()

	// this construct says focus the next InputText() box if
	//  - the terminal window is focused
	//  - AND if nothing else has been activated since last frame
	if imgui.IsWindowFocused() && !imgui.IsAnyItemActive() {
		imgui.SetKeyboardFocusHere()
	}

	if imgui.InputTextV("", &trm.input,
		imgui.InputTextFlagsEnterReturnsTrue|imgui.InputTextFlagsCallbackCompletion|imgui.InputTextFlagsCallbackHistory,
		trm.tabCompleteAndHistory) {
		trm.inputChan <- true
	}

	// add some spacing so that when we scroll to the bottom of the windw
	// it doesn't look goofy
	imgui.Spacing()

	// if output has been added to, scroll to bottom of window
	if trm.moreOutput {
		trm.moreOutput = false
		imgui.SetScrollHereY(1.0)
	}

	imgui.End()
}

func (trm *term) tabCompleteAndHistory(d imgui.InputTextCallbackData) int32 {
	switch d.EventKey() {
	case imgui.KeyTab:
		// tab completion
		b := string(d.Buffer())
		s := trm.tabCompletion.Complete(b)
		d.DeleteBytes(0, len(b))
		d.InsertBytes(0, []byte(s))
		d.MarkBufferModified()
	case imgui.KeyUpArrow:
		// previous history item
		if trm.historyIdx > -1 {
			b := string(d.Buffer())
			d.DeleteBytes(0, len(b))
			d.InsertBytes(0, []byte(trm.history[trm.historyIdx]))
			if trm.historyIdx > 0 {
				trm.historyIdx--
			}
			d.MarkBufferModified()
		}
	case imgui.KeyDownArrow:
		// next history item
		if trm.historyIdx < len(trm.history)-1 {
			b := string(d.Buffer())
			d.DeleteBytes(0, len(b))
			d.InsertBytes(0, []byte(trm.history[trm.historyIdx]))
			if trm.historyIdx < len(trm.history)-1 {
				trm.historyIdx++
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
	if trm.silenced && style != terminal.StyleError {
		return
	}

	if len(trm.output) >= outputMaxSize {
		trm.output = append(trm.output[1:], terminalOutput{style: style, text: s})
	} else {
		trm.output = append(trm.output, terminalOutput{style: style, text: s})
	}

	trm.moreOutput = true
}

// TermRead implements the terminal.Input interface
func (trm *term) TermRead(buffer []byte, prompt terminal.Prompt, events *terminal.ReadEvents) (int, error) {
	trm.prompt = prompt.Content

	// the debugger is waiting for input from the terminal but we still need to
	// service gui events in the meantime.
	for {
		select {
		case <-trm.inputChan:
			trm.input = strings.TrimSpace(trm.input)
			if trm.input != "" {
				trm.history = append(trm.history, trm.input)
				trm.historyIdx = len(trm.history) - 1
			}

			// even if term.input is the empty string we still copy it to the
			// input buffer (sending it back to the caller) because the empty
			// string might mean something

			n := len(trm.input)
			copy(buffer, trm.input+"\n")
			trm.input = ""
			return n + 1, nil

		case s := <-trm.sideChan:
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

// terminalOutput represents the lines that are printed to the terminal output
type terminalOutput struct {
	style terminal.Style
	text  string
}

func (l terminalOutput) draw() {
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
