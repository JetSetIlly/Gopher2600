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

	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/tia/video"

	"github.com/inkyblackness/imgui-go/v4"
)

const winTIAID = "TIA"
const winTIAMenu = "TIA (Video)"

type winTIA struct {
	debuggerWin

	img *SdlImgui

	popupPalette *popupPalette

	// the scope at which the editing of TIA value will take place
	deepPoking bool

	// widget dimensions
	hmoveSliderWidth            float32
	ballSizeComboDim            imgui.Vec2
	playerSizeAndCopiesComboDim imgui.Vec2
	missileSizeComboDim         imgui.Vec2
	missileCopiesComboDim       imgui.Vec2

	// footer that appear below each tab page
	footerHeight float32
}

func newWinTIA(img *SdlImgui) (window, error) {
	win := &winTIA{
		img:          img,
		popupPalette: newPopupPalette(img),
	}

	return win, nil
}

func (win *winTIA) init() {
	win.hmoveSliderWidth = imgui.FontSize() * 16
	win.ballSizeComboDim = imguiGetFrameDim("", video.BallSizes...)
	win.playerSizeAndCopiesComboDim = imguiGetFrameDim("", video.PlayerSizes...)
	win.missileSizeComboDim = imguiGetFrameDim("", video.MissileSizes...)
	win.missileCopiesComboDim = imguiGetFrameDim("", video.MissileCopies...)
}

func (win *winTIA) id() string {
	return winTIAID
}

// draw is called by service loop.
func (win *winTIA) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 31, Y: 470}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	// imgui.SetNextWindowSizeV(imgui.Vec2{X: 535, Y: 260}, imgui.ConditionAlways)
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winTIA) draw() {
	// tab-bar to switch between different "areas" of the TIA
	imgui.BeginTabBar("")
	if imgui.BeginTabItem("Playfield") {
		win.drawPlayfield()
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Player 0") {
		win.drawPlayer(0)
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Player 1") {
		win.drawPlayer(1)
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Missile 0") {
		win.drawMissile(0)
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Missile 1") {
		win.drawMissile(1)
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Ball") {
		win.drawBall()
		imgui.EndTabItem()
	}
	imgui.EndTabBar()

	win.footerHeight = imguiMeasureHeight(func() {
		imgui.Spacing()
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		imgui.Text(fmt.Sprintf("%c Changes take effect going forward and will likely not persist", fonts.GoingForward))
	})

	win.popupPalette.draw()
}
