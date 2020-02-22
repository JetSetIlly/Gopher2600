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
	"gopher2600/debugger"

	"github.com/inkyblackness/imgui-go/v2"
)

const winControlTitle = "Control"

const (
	videoCycleLabel     = "Step Video"
	cpuInstructionLabel = "Step CPU"
)

type winControl struct {
	windowManagement
	img *SdlImgui

	videoStep bool
	fps       float32

	// widget dimensions
	stepButtonDim imgui.Vec2
}

func newWinControl(img *SdlImgui) (managedWindow, error) {
	win := &winControl{
		img: img,
	}
	return win, nil
}

func (win *winControl) init() {
	win.stepButtonDim = minFrameDimension(videoCycleLabel, cpuInstructionLabel)
	win.fps = win.img.vcs.TV.GetReqFPS()
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

	imgui.SetNextWindowPosV(imgui.Vec2{645, 253}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winControlTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	dim := minFrameDimension("Run", "Halt")

	if win.img.paused {
		imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.ControlRun)
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.ControlRunHovered)
		imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.ControlRunActive)
		if imgui.ButtonV("Run", dim) {
			win.img.issueTermCommand("RUN")
		}
	} else {
		imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.ControlHalt)
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.ControlHaltHovered)
		imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.ControlHaltActive)
		if imgui.ButtonV("Halt", dim) {
			win.img.issueTermCommand("HALT")
		}
	}
	imgui.PopStyleColorV(3)

	win.drawQuantumToggle()
	imgui.Spacing()

	imgui.AlignTextToFramePadding()
	imgui.Text("Step:")
	imgui.SameLine()
	if imgui.Button("Frame") {
		win.img.issueTermCommand("STEP FRAME")
	}
	imgui.SameLine()
	if imgui.Button("Scanline") {
		win.img.issueTermCommand("STEP SCANLINE")
	}

	imgui.Spacing()

	// figuring the width of fps slider requires some care. we need to take
	// into account the width of the label and of the padding and inner
	// spacing.
	w := imgui.WindowWidth()
	w -= (imgui.CurrentStyle().FramePadding().X * 2) + (imgui.CurrentStyle().ItemInnerSpacing().X * 2)
	w -= imgui.CalcTextSize("FPS", false, 0).X
	imgui.PushItemWidth(w)

	// fps slider
	win.fps = win.img.vcs.TV.GetReqFPS()
	if imgui.SliderFloatV("FPS", &win.fps, 0.1, 100, "%.1f", 1.0) {
		win.img.vcs.TV.ReqFPS(win.fps)
	}
	imgui.PopItemWidth()

	// reset to specifcation rate on right mouse click
	if imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenDisabled) && imgui.IsMouseDown(1) {
		win.img.vcs.TV.ReqFPS(-1)
	}

	imgui.End()
}

func (win *winControl) drawQuantumToggle() {
	// make sure we know the current state of the debugger
	switch win.img.dbg.GetQuantum() {
	case debugger.QuantumVideo:
		win.videoStep = true
	default:
		win.videoStep = false
	}

	stepLabel := cpuInstructionLabel

	toggle := win.videoStep
	imgui.SameLine()
	toggleButton("quantum", &toggle, win.img.cols.TitleBgActive)
	if toggle {
		stepLabel = videoCycleLabel
		if win.videoStep != toggle {
			win.videoStep = toggle
			win.img.issueTermCommand("QUANTUM VIDEO")
		}
	} else {
		if win.videoStep != toggle {
			win.videoStep = toggle
			win.img.issueTermCommand("QUANTUM CPU")
		}
	}

	imgui.SameLine()
	if imgui.ButtonV(stepLabel, win.stepButtonDim) {
		win.img.issueTermCommand("STEP")
	}
}
