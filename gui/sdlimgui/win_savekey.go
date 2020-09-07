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
	"github.com/inkyblackness/imgui-go/v2"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/savekey"
)

const winSaveKeyTitle = "SaveKey (work in progress)"

type winSaveKey struct {
	windowManagement
	widgetDimensions

	img *SdlImgui
}

func newWinSaveKey(img *SdlImgui) (managedWindow, error) {
	win := &winSaveKey{
		img: img,
	}

	return win, nil
}

func (win *winSaveKey) init() {
	win.widgetDimensions.init()
}

func (win *winSaveKey) destroy() {
}

func (win *winSaveKey) id() string {
	return winSaveKeyTitle
}

func (win *winSaveKey) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{633, 358}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winSaveKeyTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	// oscilloscope
	imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.AudioOscBg)

	imgui.PushStyleColor(imgui.StyleColorPlotLines, imgui.Vec4{1.0, 0.0, 0.0, 1.0})
	imgui.PlotLinesV("", win.img.lz.SaveKey.SCL, 0, "SCL", savekey.TraceLo, savekey.TraceHi, imgui.Vec2{X: 512, Y: imgui.FrameHeight() * 2})

	imgui.PushStyleColor(imgui.StyleColorPlotLines, imgui.Vec4{0.0, 1.0, 0.0, 1.0})
	imgui.PlotLinesV("", win.img.lz.SaveKey.SDA, 0, "SDA", savekey.TraceLo, savekey.TraceHi, imgui.Vec2{X: 512, Y: imgui.FrameHeight() * 2})

	imgui.PopStyleColorV(3)

	imgui.End()
}
