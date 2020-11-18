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
	windowManagement
	img *SdlImgui

	rewindWaiting bool
	rewindTarget  int32

	// widget dimensions
	stepButtonDim imgui.Vec2
	runButtonDim  imgui.Vec2
	fpsLabelDim   imgui.Vec2
}

func newWinControl(img *SdlImgui) (managedWindow, error) {
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

	// fps slider
	fps := win.img.lz.TV.ReqFPS
	imgui.PushItemWidth(w)
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

	changedThisFrame := false

	// forward/backwards buttons
	imgui.SameLine()
	if imgui.Button("<") && win.rewindTarget > 0 {
		win.rewindTarget--
		win.rewindWaiting = win.img.lz.Dbg.PushRewind(int(win.rewindTarget), win.rewindTarget == e)
		changedThisFrame = true
	}
	imgui.SameLine()
	if imgui.Button(">") && win.rewindTarget < e {
		win.rewindTarget++
		win.rewindWaiting = win.img.lz.Dbg.PushRewind(int(win.rewindTarget), win.rewindTarget == e)
		changedThisFrame = true
	}

	// the < and > buttons above will affect the label of the slide below if
	// we're not careful. use either f or rewindTarget for label, depending on
	// whether either of those buttons have ben pressed this frame.
	var label string
	if changedThisFrame {
		label = fmt.Sprintf("%d", f)
	} else {
		label = fmt.Sprintf("%d", win.rewindTarget)
	}

	// rewind slider
	if imgui.SliderIntV("##rewind", &f, s, e, label) || win.rewindWaiting {
		if win.rewindTarget != f {
			win.rewindWaiting = win.img.lz.Dbg.PushRewind(int(f), f == e)
			if !win.rewindWaiting {
				win.rewindTarget = f
			}
		}
	}

	if !imgui.IsItemActive() {
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
