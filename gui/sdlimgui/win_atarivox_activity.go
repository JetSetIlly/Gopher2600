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
	"github.com/jetsetilly/gopher2600/hardware/peripherals/atarivox"
	"github.com/jetsetilly/imgui-go/v5"
)

const winAtariVoxActivityID = "AtariVox Activity"
const winAtariVoxActivityMenu = "Activity"

type winAtariVoxActivity struct {
	debuggerWin
	img      *SdlImgui
	atarivox *atarivox.AtariVox
}

func newWinAtarivox(img *SdlImgui) (window, error) {
	win := &winAtariVoxActivity{
		img: img,
	}

	return win, nil
}

func (win *winAtariVoxActivity) init() {
}

func (win *winAtariVoxActivity) id() string {
	return winAtariVoxActivityID
}

func (win *winAtariVoxActivity) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	// do not draw if atarivox is not active
	win.atarivox = win.img.cache.VCS.GetAtariVox()
	if win.atarivox == nil {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 633, Y: 358}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winAtariVoxActivity) draw() {
	style := imgui.CurrentStyle()
	dim := imgui.Vec2{
		X: max(256, imgui.WindowWidth()-((style.FramePadding().X*2)+(style.ItemInnerSpacing().X*2))),
		Y: imgui.FrameHeight() * 2}
	drawI2C(win.atarivox.SpeakJetDATA, win.atarivox.SpeakJetREADY, dim, win.img.cols, win.img)

	imgui.Spacing()
	imgui.Text(string(win.atarivox.State))
}
