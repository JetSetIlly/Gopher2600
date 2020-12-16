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

	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/gui"

	"github.com/inkyblackness/imgui-go/v2"
)

const winControlTitle = "Control"

const (
	videoCycleLabel     = "Step Video"
	cpuInstructionLabel = "Step CPU"
	runButtonLabel      = "Run"
	haltButtonLabel     = "Halt"
	fpsLabel            = "FPS"
)

type winControl struct {
	img  *SdlImgui
	open bool

	// rewinding state. target is the frame number that the user wants to
	// rewind to. pending means that the request hasn't happened yet (the
	// request will be repeated until pending is false). waiting means the
	// request has been made but has not completed yet.
	rewindTarget  int32
	rewindPending bool
	rewindWaiting bool

	// required dimensions of size sensitive widgets
	stepButtonDim imgui.Vec2
	runButtonDim  imgui.Vec2
	fpsLabelDim   imgui.Vec2
}

func newWinControl(img *SdlImgui) (window, error) {
	win := &winControl{
		img: img,
	}
	return win, nil
}

func (win *winControl) init() {
	win.stepButtonDim = imguiGetFrameDim(videoCycleLabel, cpuInstructionLabel)
	win.runButtonDim = imguiGetFrameDim(runButtonLabel, haltButtonLabel)
	win.fpsLabelDim = imguiGetFrameDim(fpsLabel)
}

func (win *winControl) destroy() {
}

func (win *winControl) id() string {
	return winControlTitle
}

func (win *winControl) isOpen() bool {
	return win.open
}

func (win *winControl) setOpen(open bool) {
	win.open = open
}

func (win *winControl) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{651, 228}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winControlTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	if win.img.state == gui.StateRunning {
		if imguiBooleanButtonV(win.img.cols, false, "Halt", win.runButtonDim) {
			win.img.term.pushCommand("HALT")
		}
	} else {
		if imguiBooleanButtonV(win.img.cols, true, "Run", win.runButtonDim) {
			win.img.term.pushCommand("RUN")
		}
	}

	win.drawQuantumToggle()
	imgui.Spacing()

	imgui.AlignTextToFramePadding()
	imgui.Text("Step:")
	imgui.SameLine()
	if imgui.Button("Frame") {
		win.img.term.pushCommand("STEP FRAME")
	}
	imgui.SameLine()
	if imgui.Button("Scanline") {
		win.img.term.pushCommand("STEP SCANLINE")
	}

	imgui.Spacing()

	// figuring the width of fps slider requires some care. we need to take
	// into account the width of the label and of the padding and inner
	// spacing.
	w := imgui.WindowWidth()
	w -= (imgui.CurrentStyle().FramePadding().X * 2) + (imgui.CurrentStyle().ItemInnerSpacing().X * 2)
	w -= win.fpsLabelDim.X
	imgui.PushItemWidth(w)

	// fps slider
	fps := win.img.lz.TV.ReqFPS
	if imgui.SliderFloatV(fpsLabel, &fps, 0.1, 100, "%.1f", 1.0) {
		win.img.lz.Dbg.PushRawEvent(func() { win.img.lz.Dbg.VCS.TV.SetFPS(fps) })
	}
	imgui.PopItemWidth()

	// reset to specification rate on right mouse click
	if imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenDisabled) && imgui.IsMouseDown(1) {
		win.img.lz.Dbg.PushRawEvent(func() { win.img.lz.Dbg.VCS.TV.SetFPS(-1) })
	}

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	// rewind sub-system
	win.drawRewind()

	imgui.End()
}

func (win *winControl) drawRewind() {
	imguiText("Rewind:")

	s := int32(win.img.lz.Rewind.Summary.Start)
	e := int32(win.img.lz.Rewind.Summary.End)
	f := int32(win.img.lz.TV.Frame)

	// we want the slider to always reflect the current frame or, if a
	// rewinding is currently taking place, it should should show the target
	// frame.
	if win.rewindWaiting {
		if f == win.rewindTarget {
			win.rewindWaiting = false
			win.rewindTarget = f
		} else {
			// rewiding is still taking place so make f equal to the target frame
			f = win.rewindTarget
		}
	} else {
		// keep track of running tv frame
		win.rewindTarget = f
	}

	// forward/backwards buttons
	imgui.SameLine()
	if imgui.Button("<") && win.rewindTarget > 0 {
		win.rewindTarget--
		win.rewindPending = win.img.lz.Dbg.PushRewind(int(win.rewindTarget), win.rewindTarget == e)
		win.rewindWaiting = true
	}
	imgui.SameLine()
	if imgui.Button(">") && win.rewindTarget < e {
		win.rewindTarget++
		win.rewindPending = win.img.lz.Dbg.PushRewind(int(win.rewindTarget), win.rewindTarget == e)
		win.rewindWaiting = true
	}

	// rewind slider
	if imgui.SliderInt("##rewind", &f, s, e) || win.rewindPending {
		win.rewindPending = win.img.lz.Dbg.PushRewind(int(f), f == e)
		win.rewindWaiting = true
		win.rewindTarget = f
	}

	// alignment information for frame number indicators below
	align := imguiRightAlignInt32(e)

	// rewind frame information
	imgui.Text(fmt.Sprintf("%d", s))
	imgui.SameLine()
	imgui.SetCursorPos(align)
	imgui.Text(fmt.Sprintf("%d", e))
}

func (win *winControl) drawQuantumToggle() {
	var videoStep bool

	// make sure we know the current state of the debugger
	if win.img.lz.Debugger.Quantum == debugger.QuantumVideo {
		videoStep = true
	}

	toggle := videoStep

	stepLabel := cpuInstructionLabel
	imgui.SameLine()
	imguiToggleButton("quantumToggle", &toggle, win.img.cols.TitleBgActive)
	if toggle {
		stepLabel = videoCycleLabel
		if videoStep != toggle {
			win.img.term.pushCommand("QUANTUM VIDEO")
		}
	} else if videoStep != toggle {
		win.img.term.pushCommand("QUANTUM CPU")
	}

	imgui.SameLine()
	if imgui.ButtonV(stepLabel, win.stepButtonDim) {
		win.img.term.pushCommand("STEP")
	}
}
