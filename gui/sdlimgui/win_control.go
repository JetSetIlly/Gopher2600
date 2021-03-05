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

	"github.com/inkyblackness/imgui-go/v4"
)

const winControlID = "Control"

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
}

func newWinControl(img *SdlImgui) (window, error) {
	win := &winControl{
		img: img,
	}
	return win, nil
}

func (win *winControl) init() {
}

func (win *winControl) id() string {
	return winControlID
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
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)

	// running
	win.drawRunButton()

	// stepping
	imgui.Spacing()
	win.drawStep()

	// frame history
	imgui.Separator()
	imgui.Spacing()
	win.drawFramHistory()

	// fps
	imgui.Separator()
	imgui.Spacing()
	win.drawFPS()

	// mouse capture button
	imguiSeparator()
	imgui.Spacing()
	win.drawMouseCapture()

	imgui.End()
}

func (win *winControl) drawRunButton() {
	runDim := imgui.Vec2{X: imguiRemainingWinWidth(), Y: imgui.FrameHeight()}
	if win.img.state == gui.StateRunning {
		if imguiBooleanButton(win.img.cols, false, "Halt", runDim) {
			win.img.term.pushCommand("HALT")
		}
	} else {
		if imguiBooleanButton(win.img.cols, true, "Run", runDim) {
			win.img.term.pushCommand("RUN")
		}
	}
}

func (win *winControl) drawStep() {
	fillWidth := imgui.Vec2{X: -1, Y: imgui.FrameHeight()}

	if imgui.BeginTable("step", 2) {
		imgui.TableSetupColumnV("step", imgui.TableColumnFlagsWidthFixed, 75, 1)
		imgui.TableNextRow()

		// step button
		imgui.TableNextColumn()

		if imgui.Button("<##Step") {
			win.img.term.pushCommand("STEP BACK")
		}
		imgui.SameLineV(0.0, 0.0)
		if imgui.ButtonV("Step", fillWidth) {
			win.img.term.pushCommand("STEP")
		}

		imgui.TableNextColumn()

		if imguiToggleButton("##quantumToggle", win.img.lz.Debugger.Quantum == debugger.QuantumVideo, win.img.cols.TitleBgActive) {
			if win.img.lz.Debugger.Quantum == debugger.QuantumVideo {
				win.img.term.pushCommand("QUANTUM INSTRUCTION")
			} else {
				win.img.term.pushCommand("QUANTUM VIDEO")
			}
		}

		imgui.SameLine()
		imgui.AlignTextToFramePadding()
		if win.img.lz.Debugger.Quantum == debugger.QuantumVideo {
			imgui.Text("Video Clock")
		} else {
			imgui.Text("CPU Instruction")
		}
		imgui.EndTable()
	}

	if imgui.BeginTable("stepframescanline", 2) {
		imgui.TableSetupColumnV("registers", imgui.TableColumnFlagsWidthFixed, imguiDivideWinWidth(2), 1)
		imgui.TableSetupColumnV("registers", imgui.TableColumnFlagsWidthFixed, imguiDivideWinWidth(2), 2)
		imgui.TableNextRow()
		imgui.TableNextColumn()

		if imgui.Button("<##Frame") {
			win.img.term.pushCommand("STEP BACK FRAME")
		}
		imgui.SameLineV(0.0, 0.0)
		if imgui.ButtonV("Frame", fillWidth) {
			win.img.term.pushCommand("STEP FRAME")
		}

		imgui.TableNextColumn()

		if imgui.Button("<##Scanline") {
			win.img.term.pushCommand("STEP BACK SCANLINE")
		}
		imgui.SameLineV(0.0, 0.0)
		if imgui.ButtonV("Scanline", fillWidth) {
			win.img.term.pushCommand("STEP SCANLINE")
		}

		imgui.EndTable()
	}
}

func (win *winControl) drawFramHistory() {
	imgui.Text("Frame History")
	imgui.Spacing()

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

	// rewind slider
	w := imguiRemainingWinWidth()
	imgui.PushItemWidth(w)
	defer imgui.PopItemWidth()

	if imgui.SliderInt("##rewind", &f, s, e) || win.rewindPending {
		win.rewindPending = !win.img.lz.Dbg.PushRewind(int(f), f == e)
		win.rewindWaiting = true
		win.rewindTarget = f
	}
}

func (win *winControl) drawFPS() {
	imgui.Text("Performance")
	imgui.Spacing()

	w := imguiRemainingWinWidth()
	imgui.PushItemWidth(w)
	defer imgui.PopItemWidth()

	// fps slider
	fps := win.img.lz.TV.ReqFPS
	if imgui.SliderFloatV("##fps", &fps, 1, 100, "%.0f fps", imgui.SliderFlagsNone) {
		win.img.lz.Dbg.PushRawEvent(func() { win.img.lz.Dbg.VCS.TV.SetFPS(fps) })
	}

	// reset to specification rate on right mouse click
	if imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenDisabled) && imgui.IsMouseDown(1) {
		win.img.lz.Dbg.PushRawEvent(func() { win.img.lz.Dbg.VCS.TV.SetFPS(-1) })
	}

	imgui.Spacing()
	if win.img.state == gui.StateRunning {
		if win.img.lz.TV.ActualFPS <= win.img.lz.TV.ReqFPS*0.95 {
			imgui.Text("running below requested FPS")
		} else if win.img.lz.TV.ActualFPS > win.img.lz.TV.ReqFPS*0.95 {
			imgui.Text("running above requested FPS")
		}
	} else if win.img.lz.TV.ReqFPS < win.img.lz.TV.Spec.FramesPerSecond {
		imgui.Text(fmt.Sprintf("below %s rate of %.0f fps", win.img.lz.TV.Spec.ID, win.img.lz.TV.Spec.FramesPerSecond))
	} else if win.img.lz.TV.ReqFPS > win.img.lz.TV.Spec.FramesPerSecond {
		imgui.Text(fmt.Sprintf("above %s rate of %.0f fps", win.img.lz.TV.Spec.ID, win.img.lz.TV.Spec.FramesPerSecond))
	} else {
		imgui.Text(fmt.Sprintf("selected %s rate of %.0f fps", win.img.lz.TV.Spec.ID, win.img.lz.TV.Spec.FramesPerSecond))
	}
}

func (win *winControl) drawMouseCapture() {
	if win.img.wm.dbgScr.isCaptured {
		imgui.AlignTextToFramePadding()
		imgui.Text("RMB or ESC to release mouse")
	} else if imgui.Button("Capture mouse") {
		win.img.setCapture(true)
	}
}
