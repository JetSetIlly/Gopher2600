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

const winTermTitle = "Terminal"

const (
	outputMaxSize = 256

	// the max number of calls to TermPrintLine() before output resumes
	// after a side-channel silence. note that this value is immediately set to
	// zero on the next call to TermRead() so missed information is unlikely
	// (maybe impossible, but I can't prove that).
	maxSideChannelSilenceDuration = 3
)

type winTerm struct {
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

	// set to a positive value when TermRead() returns from a sideChan event.
	// silences output until value decreases to zero. value will decrease
	// whenever TermPrintLine() is called or when TermRead() is called again.
	sideChannelSilence int
}

func newWinTerm(img *SdlImgui) (managedWindow, error) {
	win := &winTerm{
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

	return win, nil
}

func (win *winTerm) destroy() {
}

func (win *winTerm) id() string {
	return winTermTitle
}

// draw is called by service loop
func (win *winTerm) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{431, 381}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{534, 313}, imgui.ConditionFirstUseEver)

	imgui.PushStyleColor(imgui.StyleColorWindowBg, imgui.Vec4{0.1, 0.1, 0.2, 0.9})
	imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{2, 2})
	imgui.BeginV(winTermTitle, &win.open, 0)
	imgui.PopStyleVar()
	imgui.PopStyleColor()

	// output
	for i := range win.output {
		win.output[i].draw()
	}
	imgui.Separator()

	// prompt
	imgui.AlignTextToFramePadding()
	imgui.Text(win.prompt)
	imgui.SameLine()

	// this construct says focus the next InputText() box if
	//  - the terminal window is focused
	//  - AND if nothing else has been activated since last frame
	if imgui.IsWindowFocused() && !imgui.IsAnyItemActive() {
		imgui.SetKeyboardFocusHere()
	}

	if imgui.InputTextV("", &win.input,
		imgui.InputTextFlagsEnterReturnsTrue|imgui.InputTextFlagsCallbackCompletion|imgui.InputTextFlagsCallbackHistory,
		win.tabCompleteAndHistory) {
		win.inputChan <- true
	}

	// add some spacing so that when we scroll to the bottom of the windw
	// it doesn't look goofy
	imgui.Spacing()

	// if output has been added to, scroll to bottom of window
	if win.moreOutput {
		win.moreOutput = false
		imgui.SetScrollHereY(1.0)
	}

	imgui.End()
}

func (win *winTerm) tabCompleteAndHistory(d imgui.InputTextCallbackData) int32 {
	switch d.EventKey() {
	case imgui.KeyTab:
		// tab completion
		b := string(d.Buffer())
		s := win.tabCompletion.Complete(b)
		d.DeleteBytes(0, len(b))
		d.InsertBytes(0, []byte(s))
		d.MarkBufferModified()
	case imgui.KeyUpArrow:
		// previous history item
		if win.historyIdx > -1 {
			b := string(d.Buffer())
			d.DeleteBytes(0, len(b))
			d.InsertBytes(0, []byte(win.history[win.historyIdx]))
			if win.historyIdx > 0 {
				win.historyIdx--
			}
			d.MarkBufferModified()
		}
	case imgui.KeyDownArrow:
		// next history item
		if win.historyIdx < len(win.history)-1 {
			b := string(d.Buffer())
			d.DeleteBytes(0, len(b))
			d.InsertBytes(0, []byte(win.history[win.historyIdx]))
			if win.historyIdx < len(win.history)-1 {
				win.historyIdx++
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
func (win *winTerm) Initialise() error {
	return nil
}

// CleanUp implements the terminal.Terminal interface
func (win *winTerm) CleanUp() {
}

// RegisterTabCompletion implements the terminal.Terminal interface
func (win *winTerm) RegisterTabCompletion(tc terminal.TabCompletion) {
	win.tabCompletion = tc
}

// Silence implements the terminal.Terminal interface
func (win *winTerm) Silence(silenced bool) {
	win.silenced = silenced
}

// TermPrintLine implements the terminal.Output interface
func (win *winTerm) TermPrintLine(style terminal.Style, s string) {
	if win.sideChannelSilence > 0 {
		win.sideChannelSilence--
		return
	}

	if win.silenced && style != terminal.StyleError {
		return
	}

	if len(win.output) >= outputMaxSize {
		win.output = append(win.output[1:], terminalOutput{style: style, cols: win.img.cols, text: s})
	} else {
		win.output = append(win.output, terminalOutput{style: style, cols: win.img.cols, text: s})
	}

	win.moreOutput = true
}

// TermRead implements the terminal.Input interface
func (win *winTerm) TermRead(buffer []byte, prompt terminal.Prompt, events *terminal.ReadEvents) (int, error) {
	win.prompt = prompt.Content

	// reset sideChannelSilence
	win.sideChannelSilence = 0

	// the debugger is waiting for input from the terminal but we still need to
	// service gui events in the meantime.
	for {
		select {
		case <-win.inputChan:
			win.input = strings.TrimSpace(win.input)
			if win.input != "" {
				win.history = append(win.history, win.input)
				win.historyIdx = len(win.history) - 1
			}

			// even if term.input is the empty string we still copy it to the
			// input buffer (sending it back to the caller) because the empty
			// string might mean something

			n := len(win.input)
			copy(buffer, win.input+"\n")
			win.input = ""
			return n + 1, nil

		case s := <-win.sideChan:
			win.sideChannelSilence = maxSideChannelSilenceDuration
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
func (win *winTerm) TermReadCheck() bool {
	// report on the number of pending items in inputChan and sideChan. if
	// either of these have events waiting then that counts as true
	return len(win.inputChan) > 0 || len(win.sideChan) > 0
}

// IsInteractive implements the terminal.Input interface
func (win *winTerm) IsInteractive() bool {
	return true
}

// terminalOutput represents the lines that are printed to the terminal output
type terminalOutput struct {
	style terminal.Style
	cols  *Colors
	text  string
}

func (l terminalOutput) draw() {
	switch l.style {
	case terminal.StyleInput:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleInput)

	case terminal.StyleHelp:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleHelp)

	case terminal.StylePromptCPUStep:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStylePromptCPUStep)

	case terminal.StylePromptVideoStep:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStylePromptVideoStep)

	case terminal.StylePromptConfirm:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStylePromptConfirm)

	case terminal.StyleFeedback:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleFeedback)

	case terminal.StyleCPUStep:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleCPUStep)

	case terminal.StyleVideoStep:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleVideoStep)

	case terminal.StyleInstrument:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleInstrument)

	case terminal.StyleFeedbackNonInteractive:
		// just use regular feedback style for this
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleFeedback)

	case terminal.StyleError:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleError)
	}

	imgui.Text(l.text)
	imgui.PopStyleColor()
}
