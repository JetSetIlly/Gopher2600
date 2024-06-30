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
	"fmt"
	"os"
	"strings"

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources/unique"

	"github.com/inkyblackness/imgui-go/v4"
)

const winTermID = "Terminal"

const outputMaxSize = 4096

type winTerm struct {
	debuggerWin

	img *SdlImgui

	term *term

	input      string
	prompt     terminal.Prompt
	output     []terminalOutput
	moreOutput bool

	history    []string
	historyIdx int

	// height of input line at bottom of window
	inputLineHeight float32

	// preferences
	wrap bool

	focusOnInput bool
}

func newWinTerm(img *SdlImgui) (window, error) {
	win := &winTerm{
		img:          img,
		term:         img.term,
		historyIdx:   -1,
		wrap:         true,
		focusOnInput: true,
	}

	return win, nil
}

func (win *winTerm) init() {
}

func (win *winTerm) id() string {
	return winTermID
}

func (win *winTerm) debuggerDraw() bool {
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

			if win.img.prefs.terminalOnError.Get().(bool) && t.style == terminal.StyleError {
				win.debuggerSetOpen(true)
			}

			win.moreOutput = true
		default:
			done = true
		}
	}

	// window open check must happen *after* channel polling
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{431, 381}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{623, 322}, imgui.ConditionFirstUseEver)

	imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.TermBackground)
	imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{2, 2})
	defer imgui.PopStyleVar()
	defer imgui.PopStyleColor()

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}
	imgui.End()

	win.debuggerGeom.update()

	return true
}

func (win *winTerm) draw() {
	imgui.PushFont(win.img.fonts.terminal)
	defer imgui.PopFont()

	height := imguiRemainingWinHeight() - win.inputLineHeight
	if imgui.BeginChildV("scrollback", imgui.Vec2{X: 0, Y: height}, false, 0) {
		win.focusOnInput = win.focusOnInput || imgui.IsItemActive() || (imgui.IsWindowHovered() && imgui.IsMouseReleased(0))

		// only draw elements that will be visible
		var clipper imgui.ListClipper
		clipper.Begin(len(win.output))
		for clipper.Step() {
			for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
				win.output[i].draw(win.wrap)
			}
		}

		// if output has been added to, scroll to bottom of window
		if win.moreOutput {
			win.moreOutput = false
			imgui.SetScrollHereY(0.0)
		}

	}
	imgui.EndChild()

	// context menu for scrollback area
	if imgui.BeginPopupContextItem() {
		imgui.Checkbox("Word wrap", &win.wrap)
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		if imgui.Selectable("Clear terminal") {
			win.output = win.output[:0]
		}
		if imgui.Selectable("Save output to file") {
			win.saveOutput()
		}
		imgui.EndPopup()
	}

	// measure command line
	win.inputLineHeight = imguiMeasureHeight(func() {
		imgui.Separator()
		imgui.Spacing()

		// show prompt. if emulation is not running we show the prompt as it
		// was supplied to us by the emulation (via the TermRead() function).
		//
		// if the emulation is running then the TermRead() prompt isn't good
		// enough because it won't be updated until the next TermRead(), which
		// won't happen until the emulation pauses
		//
		// to indicate movement therefore, we use the LastResult as retreived
		// from the debugger by the lazy system (this is what is shown in the
		// CPU window)
		//
		// note that we don't use LastResult all the time because that will
		// show the disassembly for the *previous* instruction executed.
		//
		// the prompt in the terminal should show the disassembly for the
		// instruction the PC is *currently* on. in other words, the
		// disassembly for the inesturction to be executed *next*
		if win.img.dbg.State() == govern.Running {
			res := win.img.cache.Dbg.LiveDisasmEntry
			imgui.Text(res.String())
			if !win.img.cache.Dbg.LiveDisasmEntry.Result.Final {
				imgui.SameLine()
				imgui.Text(res.Cycles())
			}
		} else {
			imgui.Text(win.prompt.Content)

			// additional yield information
			if !win.prompt.CoProcYield.Type.Normal() {
				imgui.SameLine()
				imgui.Text(fmt.Sprintf("(%s)", win.prompt.CoProcYield.Type))
				if win.prompt.CoProcYield.Error != nil {
					imguiTooltip(func() {
						imgui.Text("Coprocessor Yield")
						imgui.Spacing()
						imgui.Separator()
						imgui.Spacing()
						imgui.Text(fmt.Sprintf("%s: %s", win.prompt.CoProcYield.Type, win.prompt.CoProcYield.Error.Error()))
					}, true, true)
				}
			}
		}

		// chevron indicator
		imgui.Spacing()
		imgui.AlignTextToFramePadding()
		imgui.Text(fmt.Sprintf("%c", fonts.TermPrompt))
		imgui.SameLine()

		// command line to be the width of the window
		imgui.PushItemWidth(-1)

		// keyboard focus on InputText widge
		if win.focusOnInput {
			imgui.SetKeyboardFocusHere()
			win.focusOnInput = false
		}

		if imgui.InputTextV("", &win.input,
			imgui.InputTextFlagsEnterReturnsTrue|imgui.InputTextFlagsCallbackCompletion|imgui.InputTextFlagsCallbackHistory,
			win.tabCompleteAndHistory) {

			win.input = strings.TrimSpace(win.input)

			// send input to inputChan even if it is the empty string because
			// the empty string might mean something to the received (it does)
			win.term.inputChan <- win.input

			// only add input to history if it is not empty
			if win.input != "" {
				// only add if input is not the same as the last history entry
				if len(win.history) == 0 || win.input != win.history[len(win.history)-1] {
					win.history = append(win.history, win.input)
				}
				win.historyIdx = len(win.history) - 1
			}

			win.input = ""

			// pressing enter unfocuses the InpuText() widget. we get around this by
			// setting focusOnInput to true, meaning that we can continue our typing
			// operation.
			win.focusOnInput = true
		}

		imgui.PopItemWidth()
	})
}

func (win *winTerm) saveOutput() {
	fn := unique.Filename("terminal", "")
	f, err := os.Create(fn)
	if err != nil {
		win.output = append(win.output, terminalOutput{
			style: terminal.StyleError,
			cols:  win.img.cols,
			text:  "could not save terminal output",
		})
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Logf(logger.Allow, "sdlimgui", "error saving terminal contents: %v", err)
		}
	}()

	for _, o := range win.output {
		f.Write([]byte(o.text))
		f.Write([]byte("\n"))
	}

	win.output = append(win.output, terminalOutput{
		style: terminal.StyleFeedback,
		cols:  win.img.cols,
		text:  fmt.Sprintf("terminal output saved to %s", fn),
	})
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
			if win.historyIdx < len(win.history)-1 {
				win.historyIdx++
			}
			d.DeleteBytes(0, len(b))
			d.InsertBytes(0, []byte(win.history[win.historyIdx]))
		} else {
			b := string(d.Buffer())
			d.DeleteBytes(0, len(b))
		}
		d.MarkBufferModified()
	}
	return 0
}

// terminalOutput represents the lines that are printed to the terminal output.
type terminalOutput struct {
	style terminal.Style
	cols  *imguiColors
	text  string
}

func (l terminalOutput) draw(wrap bool) {
	switch l.style {
	case terminal.StyleEcho:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleEcho)

	case terminal.StyleHelp:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleHelp)

	case terminal.StyleFeedback:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleFeedback)

	case terminal.StyleFeedbackSecondary:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleFeedbackSecondary)

	case terminal.StyleInstructionStep:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleCPUStep)

	case terminal.StyleSubStep:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleVideoStep)

	case terminal.StyleInstrument:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleInstrument)

	case terminal.StyleError:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleError)

	case terminal.StyleLog:
		imgui.PushStyleColor(imgui.StyleColorText, l.cols.TermStyleLog)
	}

	// text wrap for window
	if wrap {
		imgui.PushTextWrapPosV(imgui.WindowWidth())
		defer imgui.PopTextWrapPos()
	}

	imgui.Text(l.text)

	imgui.PopStyleColor()
}
