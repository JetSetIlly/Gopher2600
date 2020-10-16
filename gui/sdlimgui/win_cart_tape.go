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

	"github.com/inkyblackness/imgui-go/v2"
)

const winCartTapeTitle = "Cassette Tape"

type winCartTape struct {
	windowManagement

	img *SdlImgui
}

func newWinCartTape(img *SdlImgui) (managedWindow, error) {
	win := &winCartTape{
		img: img,
	}

	return win, nil
}

func (win *winCartTape) init() {
}

func (win *winCartTape) destroy() {
}

func (win *winCartTape) id() string {
	return winCartTapeTitle
}

func (win *winCartTape) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.HasTapeBus {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{633, 358}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winCartTapeTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	// counter information
	imguiText("Counter")
	counter := fmt.Sprintf("%8d", win.img.lz.Cart.TapeState.Counter)
	if imguiDecimalInput("##counter", !win.img.paused, 8, &counter) {
		win.img.lz.Dbg.PushRawEvent(func() {
			c, err := strconv.ParseInt(counter, 10, 64)
			if err == nil {
				win.img.lz.Cart.TapeBus.SetTapeCounter(int(c))
			}
		})
	}
	imgui.SameLine()
	imgui.Text("/")
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("%8d", win.img.lz.Cart.TapeState.MaxCounter))

	// time information
	imgui.Text(fmt.Sprintf("%.02fs", win.img.lz.Cart.TapeState.Time))
	imgui.SameLine()
	imgui.Text("/")
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("%.02fs", win.img.lz.Cart.TapeState.MaxTime))

	// oscilloscope
	imgui.Spacing()
	w := imgui.WindowWidth()
	w -= (imgui.CurrentStyle().FramePadding().X * 2) + (imgui.CurrentStyle().ItemInnerSpacing().X * 2)
	imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.AudioOscBg)
	imgui.PushStyleColor(imgui.StyleColorPlotLines, win.img.cols.AudioOscLine)
	imgui.PlotLinesV("", win.img.lz.Cart.TapeState.Data, 0, "", -1.0, 1.0,
		imgui.Vec2{X: w, Y: imgui.FrameHeight() * 2})
	imgui.PopStyleColorV(2)
	imgui.Spacing()

	// rewind button
	if imgui.Button("Rewind") {
		_ = win.img.lz.Cart.TapeBus.Rewind()
	}

	imgui.End()
}
