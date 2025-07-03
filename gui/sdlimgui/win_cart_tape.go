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
	"strconv"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/imgui-go/v5"
)

const winCartTapeID = "Cassette Tape"

type winCartTape struct {
	debuggerWin

	img *SdlImgui
}

func newWinCartTape(img *SdlImgui) (window, error) {
	win := &winCartTape{
		img: img,
	}

	return win, nil
}

func (win *winCartTape) init() {
}

func (win *winCartTape) id() string {
	return winCartTapeID
}

func (win *winCartTape) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	// do not open window if there is no cartridge tape bus available
	bus := win.img.cache.VCS.Mem.Cart.GetTapeBus()
	if bus == nil {
		return false
	}
	ok, tape := bus.GetTapeState()
	if !ok {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 539, Y: 168}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw(tape)
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCartTape) draw(tape mapper.CartTapeState) {
	// counter information
	imguiLabel("Counter")
	counter := fmt.Sprintf("%8d", tape.Counter)
	if imguiDecimalInput("##counter", 8, &counter) {
		win.img.dbg.PushFunction(func() {
			c, err := strconv.ParseInt(counter, 10, 64)
			if err == nil {
				win.img.dbg.VCS().Mem.Cart.GetTapeBus().SetTapeCounter(int(c))
			}
		})
	}
	imgui.SameLine()
	imgui.Text("/")
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("%8d", tape.MaxCounter))

	// time information
	imgui.Text(fmt.Sprintf("%.02fs", tape.Time))
	imgui.SameLine()
	imgui.Text("/")
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("%.02fs", tape.MaxTime))

	// oscilloscope
	imgui.Spacing()
	w := imgui.WindowWidth()
	w -= (imgui.CurrentStyle().FramePadding().X * 2) + (imgui.CurrentStyle().ItemInnerSpacing().X * 2)
	imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.AudioOscBg)
	imgui.PushStyleColor(imgui.StyleColorPlotLines, win.img.cols.AudioOscLine)
	imgui.PlotLinesV("##tapeoscilloscope", tape.Data, 0, "", -1.0, 1.0,
		imgui.Vec2{X: w, Y: imgui.FrameHeight() * 2})
	imgui.PopStyleColorV(2)
	imgui.Spacing()

	// tape slider
	c := int32(tape.Counter)
	if imgui.SliderIntV("##counterslider", &c, 0, int32(tape.MaxCounter), "", imgui.SliderFlagsNone) {
		win.img.dbg.PushFunction(func() {
			win.img.dbg.VCS().Mem.Cart.GetTapeBus().SetTapeCounter(int(c))
		})
	}

	// rewind button
	imgui.SameLine()
	if imgui.Button("Rewind") {
		win.img.dbg.PushFunction(func() {
			win.img.dbg.VCS().Mem.Cart.GetTapeBus().Rewind()
		})
	}
}
