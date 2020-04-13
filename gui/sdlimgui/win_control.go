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
	"github.com/jetsetilly/gopher2600/debugger"

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

	if win.img.paused {
		imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.ControlRun)
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.ControlRunHovered)
		imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.ControlRunActive)
		if imgui.ButtonV(runButtonLabel, win.runButtonDim) {
			win.img.term.pushCommand("RUN")
		}
	} else {
		imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.ControlHalt)
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.ControlHaltHovered)
		imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.ControlHaltActive)
		if imgui.ButtonV(haltButtonLabel, win.runButtonDim) {
			win.img.term.pushCommand("HALT")
		}
	}
	imgui.PopStyleColorV(3)

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
	fps := win.img.lazy.TV.ReqFPS
	imgui.PushItemWidth(w)
	if imgui.SliderFloatV(fpsLabel, &fps, 0.1, 100, "%.1f", 1.0) {
		win.img.lazy.Dbg.PushRawEvent(func() { win.img.lazy.Dbg.SetFPS(fps) })
	}
	imgui.PopItemWidth()

	// reset to specifcation rate on right mouse click
	if imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenDisabled) && imgui.IsMouseDown(1) {
		win.img.lazy.Dbg.PushRawEvent(func() { win.img.lazy.Dbg.SetFPS(-1) })
	}

	imgui.End()
}

func (win *winControl) drawQuantumToggle() {
	var videoStep bool

	// make sure we know the current state of the debugger
	if win.img.lazy.Debugger.Quantum == debugger.QuantumVideo {
		videoStep = true
	}

	toggle := videoStep

	stepLabel := cpuInstructionLabel
	imgui.SameLine()
	imguiToggleButton("quantumToggle", &toggle, win.img.cols.TitleBgActive)
	if toggle {
		stepLabel = videoCycleLabel
		if videoStep != toggle {
			videoStep = toggle
			win.img.term.pushCommand("QUANTUM VIDEO")
		}
	} else {
		if videoStep != toggle {
			videoStep = toggle
			win.img.term.pushCommand("QUANTUM CPU")
		}
	}

	imgui.SameLine()
	if imgui.ButtonV(stepLabel, win.stepButtonDim) {
		win.img.term.pushCommand("STEP")
	}
}
