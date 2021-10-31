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
	"github.com/jetsetilly/gopher2600/hardware/tia/video"

	"github.com/inkyblackness/imgui-go/v4"
)

const winTIAID = "TIA"

type winTIA struct {
	img  *SdlImgui
	open bool

	popupPalette *popupPalette
	strobe       int32

	// the scope at which the editing of TIA value will take place
	deepPoking bool

	// widget dimensions
	hmoveSliderWidth            float32
	ballSizeComboDim            imgui.Vec2
	playerSizeAndCopiesComboDim imgui.Vec2
	missileSizeComboDim         imgui.Vec2
	missileCopiesComboDim       imgui.Vec2

	// scope selection height
	scopeHeight float32
}

func newWinTIA(img *SdlImgui) (window, error) {
	win := &winTIA{
		img:          img,
		popupPalette: newPopupPalette(img),
		strobe:       -1,
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

func (win *winTIA) isOpen() bool {
	return win.open
}

func (win *winTIA) setOpen(open bool) {
	win.open = open
}

// draw is called by service loop.
func (win *winTIA) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 21, Y: 491}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 531, Y: 241}, imgui.ConditionFirstUseEver)
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsNoResize)

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

	win.drawPersistenceControl()

	imgui.End()

	win.popupPalette.draw()
}
