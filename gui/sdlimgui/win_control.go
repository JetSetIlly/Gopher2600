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

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui/fonts"

	"github.com/jetsetilly/imgui-go/v5"
)

const winControlID = "Control"

type winControl struct {
	debuggerWin

	img *SdlImgui

	repeatID         string
	repeatTime       time.Time
	repeatFPSLimiter atomic.Value // bool
}

func newWinControl(img *SdlImgui) (window, error) {
	win := &winControl{
		img: img,
	}
	win.debuggerGeom.noFocusTracking = true
	return win, nil
}

func (win *winControl) init() {
}

func (win *winControl) id() string {
	return winControlID
}

func (win *winControl) repeatButton(id string, f func()) {
	win.repeatButtonV(id, f, imgui.Vec2{})
}

func (win *winControl) repeatButtonV(id string, f func(), fill imgui.Vec2) {
	imgui.ButtonV(id, fill)
	if imgui.IsItemActive() {
		if id != win.repeatID {
			win.img.dbg.PushFunction(func() {
				v := win.img.dbg.VCS().TV.SetFPSLimit(false)
				win.repeatFPSLimiter.Store(v)
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
		win.img.dbg.PushFunction(func() {
			v := win.repeatFPSLimiter.Load().(bool)
			win.img.dbg.VCS().TV.SetFPSLimit(v)
		})
	}
}

func (win *winControl) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 699, Y: 45}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSize(imgui.Vec2{X: imguiTextWidth(36), Y: -1})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winControl) draw() {
	// running
	win.drawRunButton()

	// stepping
	imgui.Spacing()
	win.drawStep()

	// fps
	imguiSeparator()
	win.drawFPS()

	// mouse capture button
	imguiSeparator()
	win.drawMouseCapture()
}

func (win *winControl) drawRunButton() {
	dim := imgui.Vec2{X: imguiRemainingWinWidth(), Y: imgui.FrameHeight()}
	if win.img.dbg.State() == govern.Running {
		if imguiColourButton(win.img.cols.False, fmt.Sprintf("%c Halt", fonts.Halt), dim) {
			win.img.term.pushCommand("HALT")
		}
	} else {
		if imguiColourButton(win.img.cols.True, fmt.Sprintf("%c Run", fonts.Run), dim) {
			win.img.term.pushCommand("RUN")
		}
	}
}

func (win *winControl) drawStep() {
	fillWidth := imgui.Vec2{X: -1, Y: imgui.FrameHeight()}

	if imgui.BeginTable("step", 2) {
		imgui.TableSetupColumnV("stepframescanline0", imgui.TableColumnFlagsWidthFixed, imguiDivideWinWidth(2), 0)
		imgui.TableSetupColumnV("stepframescanline1", imgui.TableColumnFlagsWidthFixed, imguiDivideWinWidth(2), 1)
		imgui.TableNextRow()

		// step button
		imgui.BeginGroup()
		imgui.TableNextColumn()
		win.repeatButton(fmt.Sprintf("%c ##Step", fonts.BackArrowDouble), func() {
			if win.img.dbg.State() == govern.Paused {
				win.img.term.pushCommand("STEP BACK")
			}
		})

		imgui.SameLineV(0.0, 0.0)
		win.repeatButtonV("Step", func() {
			if win.img.dbg.State() == govern.Paused {
				win.img.term.pushCommand("STEP")
			}
		}, fillWidth)
		imgui.EndGroup()

		imgui.TableNextColumn()

		imgui.PushItemWidth(-1)
		if imgui.BeginComboV("##quantum", win.img.dbg.Quantum().String(), imgui.ComboFlagsNone) {
			if imgui.Selectable(govern.QuantumInstruction.String()) {
				win.img.term.pushCommand("QUANTUM INSTRUCTION")
			}
			if imgui.Selectable(govern.QuantumCycle.String()) {
				win.img.term.pushCommand("QUANTUM CYCLE")
			}
			if imgui.Selectable(govern.QuantumClock.String()) {
				win.img.term.pushCommand("QUANTUM CLOCK")
			}
			imgui.EndCombo()
		}
		imgui.PopItemWidth()

		imgui.EndTable()
	}

	win.repeatButtonV(fmt.Sprintf("%c Step Over", fonts.StepOver), func() {
		if win.img.dbg.State() == govern.Paused {
			win.img.term.pushCommand("STEP OVER")
		}
	}, fillWidth)

	if imgui.BeginTable("stepframescanline", 2) {
		imgui.TableSetupColumnV("stepframescanline0", imgui.TableColumnFlagsWidthFixed, imguiDivideWinWidth(2), 0)
		imgui.TableSetupColumnV("stepframescanline1", imgui.TableColumnFlagsWidthFixed, imguiDivideWinWidth(2), 1)
		imgui.TableNextRow()
		imgui.TableNextColumn()

		imgui.BeginGroup()
		win.repeatButton(fmt.Sprintf("%c ##Frame", fonts.UpArrowDouble), func() {
			if win.img.dbg.State() == govern.Paused {
				win.img.term.pushCommand("STEP BACK FRAME")
			}
		})
		imgui.SameLineV(0.0, 0.0)
		win.repeatButtonV("Frame", func() {
			if win.img.dbg.State() == govern.Paused {
				win.img.term.pushCommand("STEP FRAME")
			}
		}, fillWidth)
		imgui.EndGroup()

		imgui.TableNextColumn()

		imgui.BeginGroup()
		win.repeatButton(fmt.Sprintf("%c ##Scanline", fonts.UpArrow), func() {
			if win.img.dbg.State() == govern.Paused {
				win.img.term.pushCommand("STEP BACK SCANLINE")
			}
		})
		imgui.SameLineV(0.0, 0.0)
		win.repeatButtonV("Scanline", func() {
			if win.img.dbg.State() == govern.Paused {
				win.img.term.pushCommand("STEP SCANLINE")
			}
		}, fillWidth)
		imgui.EndGroup()

		imgui.EndTable()
	}
}

func (win *winControl) drawFPS() {
	ideal := win.img.dbg.VCS().TV.GetIdealFPS()
	actual, _ := win.img.dbg.VCS().TV.GetActualFPS()
	frameInfo := win.img.cache.TV.GetFrameInfo()

	imgui.Text("Performance")
	imgui.Spacing()

	w := imguiRemainingWinWidth()
	imgui.PushItemWidth(w)
	defer imgui.PopItemWidth()

	// fps slider
	if imgui.SliderFloatV("##fps", &ideal, 1, 100, "%.0f fps", imgui.SliderFlagsNone) {
		win.img.dbg.PushFunction(func() { win.img.dbg.VCS().TV.SetFPS(ideal) })
	}

	// reset to specification rate on right mouse click
	if imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenDisabled) && imgui.IsMouseDown(1) {
		win.img.dbg.PushFunction(func() { win.img.dbg.VCS().TV.SetFPS(-1) })
	}

	imgui.Spacing()
	if win.img.dbg.State() == govern.Running {
		if actual <= ideal*0.95 {
			imgui.Text("running below requested FPS")
		} else if actual > ideal*1.05 {
			imgui.Text("running above requested FPS")
		} else {
			imgui.Text("running at requested FPS")
		}
	} else if ideal < frameInfo.Spec.RefreshRate*0.95 {
		imgui.Text(fmt.Sprintf("below ideal frequency of %.0fHz", frameInfo.Spec.RefreshRate))
	} else if ideal > frameInfo.Spec.RefreshRate*1.05 {
		imgui.Text(fmt.Sprintf("above ideal frequency of %.0fHz", frameInfo.Spec.RefreshRate))
	} else {
		imgui.Text(fmt.Sprintf("ideal frequency %.0fHz", frameInfo.Spec.RefreshRate))
	}
}

func (win *winControl) drawMouseCapture() {
	imgui.BeginGroup()
	imgui.AlignTextToFramePadding()
	imgui.Text(string(fonts.Mouse))
	imgui.SameLine()
	if win.img.wm.dbgScr.isCaptured {
		imgui.AlignTextToFramePadding()
		label := "RMB to release input"
		if win.img.dbg.State() == govern.Running {
			label = "RMB to halt & release input"
		}
		imgui.Text(label)
	} else {
		label := "Capture input & run"
		if win.img.dbg.State() == govern.Running {
			label = "Capture input & continue"
		}
		if imgui.Button(label) {
			win.img.setCapturedRunning(true)
		}
	}
	imgui.EndGroup()
}
