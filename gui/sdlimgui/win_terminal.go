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
	"strings"

	"github.com/jetsetilly/gopher2600/debugger/terminal"

	"github.com/inkyblackness/imgui-go/v2"
)

const winTermTitle = "Terminal"

const outputMaxSize = 512

type winTerm struct {
	windowManagement
	img  *SdlImgui
	term *term

	input      string
	prompt     string
	output     []terminalOutput
	moreOutput bool

	history    []string
	historyIdx int
}

func newWinTerm(img *SdlImgui) (managedWindow, error) {
	win := &winTerm{
		img:        img,
		term:       img.term,
		historyIdx: -1,
	}

	return win, nil
}

func (win *winTerm) init() {
}

func (win *winTerm) destroy() {
}

func (win *winTerm) id() string {
	return winTermTitle
}

func (win *winTerm) draw() {
	done := false
	for !done {
		// check for channel activity before we do anything
		select {
		case p := <-win.term.promptChan:
			win.prompt = p

		case t := <-win.term.outputChan:
			t.cols = win.img.cols
			if len(win.output) >= outputMaxSize {
				win.output = append(win.output[1:], t)
			} else {
				win.output = append(win.output, t)
			}

			win.moreOutput = true
		default:
			done = true
		}
	}

	// window open check must happen *after* channel polling
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{431, 381}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{534, 313}, imgui.ConditionFirstUseEver)

	imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.TermBackground)
	imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{2, 2})
	imgui.BeginV(winTermTitle, &win.open, 0)
	imgui.PopStyleVar()
	imgui.PopStyleColor()

	// only draw elements that will be visible
	var clipper imgui.ListClipper
	clipper.Begin(len(win.output))
	for clipper.Step() {
		for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
			win.output[i].draw()
		}
	}

	if len(win.output) > 0 {
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
	}

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

	// draw command input box
	imgui.PushItemWidth(imgui.WindowWidth() - imgui.CursorPosX())
	imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.TermBackground)
	if imgui.InputTextV("", &win.input,
		imgui.InputTextFlagsEnterReturnsTrue|imgui.InputTextFlagsCallbackCompletion|imgui.InputTextFlagsCallbackHistory,
		win.tabCompleteAndHistory) {

		win.input = strings.TrimSpace(win.input)

		// send input to inputChan even if it is the empty string because
		// the empty string might mean something to the received (it does)
		win.term.inputChan <- win.input

		// only add input to history if it is not empty
		if win.input != "" {
			win.history = append(win.history, win.input)
			win.historyIdx = len(win.history) - 1
		}

		win.input = ""
	}
	imgui.PopStyleColor()
	imgui.PopItemWidth()

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
		s := win.term.tabCompletion.Complete(b)
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

// terminalOutput represents the lines that are printed to the terminal output
type terminalOutput struct {
	style terminal.Style
	cols  *imguiColors
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
