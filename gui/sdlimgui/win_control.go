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
	"sync/atomic"
	"time"

	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/gui/fonts"

	"github.com/inkyblackness/imgui-go/v4"
)

const winControlID = "Control"

type winControl struct {
	img  *SdlImgui
	open bool

	repeatID     string
	repeatTime   time.Time
	repeatFPSCap atomic.Value // bool
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

func (win *winControl) repeatButton(id string, f func()) {
	win.repeatButtonV(id, f, imgui.Vec2{})
}

func (win *winControl) repeatButtonV(id string, f func(), fill imgui.Vec2) {
	imgui.ButtonV(id, fill)
	if imgui.IsItemActive() {
		if id != win.repeatID {
			win.img.dbg.PushRawEvent(func() {
				v := win.img.vcs.TV.SetFPSCap(false)
				win.repeatFPSCap.Store(v)
			})
			win.repeatID = id
			win.repeatTime = time.Now()
			f()
			return
		}

		dur := time.Since(win.repeatTime)
		if dur > 5e+8 { // half a second in nanoseconds
			f()
		}
	} else if imgui.IsItemDeactivated() {
		win.repeatID = ""
		win.img.dbg.PushRawEvent(func() {
			v := win.repeatFPSCap.Load().(bool)
			win.img.vcs.TV.SetFPSCap(v)
		})
	}
}

func (win *winControl) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{699, 45}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)

	// running
	win.drawRunButton()

	// stepping
	imgui.Spacing()
	win.drawStep()

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
	if win.img.emulation.State() == emulation.Running {
		if imguiColourButton(win.img.cols.False, fmt.Sprintf("%c Halt", fonts.Halt), runDim) {
			win.img.term.pushCommand("HALT")
		}
	} else {
		if imguiColourButton(win.img.cols.True, fmt.Sprintf("%c Run", fonts.Run), runDim) {
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

		icon := fonts.BackInstruction
		if win.img.lz.Debugger.Quantum == debugger.QuantumClock {
			icon = fonts.BackClock
		}

		win.repeatButton(fmt.Sprintf("%c ##Step", icon), func() {
			win.img.term.pushCommand("STEP BACK")
		})

		imgui.SameLineV(0.0, 0.0)
		win.repeatButtonV("Step", func() {
			win.img.term.pushCommand("STEP")
		}, fillWidth)

		imgui.TableNextColumn()

		if imguiToggleButton("##quantumToggle", win.img.lz.Debugger.Quantum == debugger.QuantumClock, win.img.cols.TitleBgActive) {
			if win.img.lz.Debugger.Quantum == debugger.QuantumClock {
				win.img.term.pushCommand("QUANTUM INSTRUCTION")
			} else {
				win.img.term.pushCommand("QUANTUM CLOCK")
			}
		}

		imgui.SameLine()
		imgui.AlignTextToFramePadding()
		if win.img.lz.Debugger.Quantum == debugger.QuantumClock {
			imgui.Text("Colour Clock")
		} else {
			imgui.Text("CPU Instruction")
		}

		imgui.EndTable()
	}

	win.repeatButtonV(fmt.Sprintf("%c Step Over", fonts.StepOver), func() {
		win.img.term.pushCommand("STEP OVER")
	}, fillWidth)

	if imgui.BeginTable("stepframescanline", 2) {
		imgui.TableSetupColumnV("registers", imgui.TableColumnFlagsWidthFixed, imguiDivideWinWidth(2), 1)
		imgui.TableSetupColumnV("registers", imgui.TableColumnFlagsWidthFixed, imguiDivideWinWidth(2), 2)
		imgui.TableNextRow()
		imgui.TableNextColumn()

		win.repeatButton(fmt.Sprintf("%c ##Frame", fonts.BackFrame), func() {
			win.img.term.pushCommand("STEP BACK FRAME")
		})
		imgui.SameLineV(0.0, 0.0)
		win.repeatButtonV("Frame", func() {
			win.img.term.pushCommand("STEP FRAME")
		}, fillWidth)

		imgui.TableNextColumn()

		win.repeatButton(fmt.Sprintf("%c ##Scanline", fonts.BackScanline), func() {
			win.img.term.pushCommand("STEP BACK SCANLINE")
		})
		imgui.SameLineV(0.0, 0.0)
		win.repeatButtonV("Scanline", func() {
			win.img.term.pushCommand("STEP SCANLINE")
		}, fillWidth)

		imgui.EndTable()
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
		win.img.dbg.PushRawEvent(func() { win.img.vcs.TV.SetFPS(fps) })
	}

	// reset to specification rate on right mouse click
	if imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenDisabled) && imgui.IsMouseDown(1) {
		win.img.dbg.PushRawEvent(func() { win.img.vcs.TV.SetFPS(-1) })
	}

	imgui.Spacing()
	if win.img.emulation.State() == emulation.Running {
		if win.img.lz.TV.ActualFPS <= win.img.lz.TV.ReqFPS*0.95 {
			imgui.Text("running below requested FPS")
		} else if win.img.lz.TV.ActualFPS > win.img.lz.TV.ReqFPS*1.05 {
			imgui.Text("running above requested FPS")
		} else {
			imgui.Text("running at requested FPS")
		}
	} else if win.img.lz.TV.ReqFPS < win.img.lz.TV.FrameInfo.Spec.RefreshRate*0.95 {
		imgui.Text(fmt.Sprintf("below ideal frequency of %.0fHz", win.img.lz.TV.FrameInfo.Spec.RefreshRate))
	} else if win.img.lz.TV.ReqFPS > win.img.lz.TV.FrameInfo.Spec.RefreshRate*1.05 {
		imgui.Text(fmt.Sprintf("above ideal frequency of %.0fHz", win.img.lz.TV.FrameInfo.Spec.RefreshRate))
	} else {
		imgui.Text(fmt.Sprintf("ideal frequency %.0fHz", win.img.lz.TV.FrameInfo.Spec.RefreshRate))
	}
}

func (win *winControl) drawMouseCapture() {
	imgui.AlignTextToFramePadding()
	imgui.Text(string(fonts.Mouse))
	imgui.SameLine()
	if win.img.wm.dbgScr.isCaptured {
		imgui.AlignTextToFramePadding()
		label := "RMB to release input"
		if win.img.emulation.State() == emulation.Running {
			label = "RMB to halt & release input"
		}
		imgui.Text(label)
	} else {
		label := "Capture input & run"
		if win.img.emulation.State() == emulation.Running {
			label = "Capture input & continue"
		}
		if imgui.Button(label) {
			win.img.setCapture(true)
			win.img.term.pushCommand("RUN")
		}
	}
}
