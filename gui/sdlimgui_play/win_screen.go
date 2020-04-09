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

package sdlimgui_play

import (
	"github.com/inkyblackness/imgui-go/v2"
)

const winScreenTitle = "TV Screen"

type winScreen struct {
	img *SdlImguiPlay
	scr *screen

	// is screen currently pointed at
	isHovered bool

	// the tv screen has captured mouse input
	isCaptured bool
}

func newWinScreen(img *SdlImguiPlay) (*winScreen, error) {
	win := &winScreen{
		img:        img,
		scr:        img.screen,
		isCaptured: false,
	}

	return win, nil
}

func (win *winScreen) id() string {
	return winScreenTitle
}

// draw is called by service loop
func (win *winScreen) draw() {
	imgui.PushStyleVarFloat(imgui.StyleVarWindowBorderSize, 0.0)
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.Vec2{0.0, 0.0})

	imgui.SetNextWindowPosV(imgui.Vec2{0, 0}, 0, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{win.img.screen.scaledWidth(), win.img.screen.scaledHeight()}, 0)
	imgui.BeginV(winScreenTitle, nil, imgui.WindowFlagsNoDecoration|imgui.WindowFlagsNoSavedSettings)

	// actual display
	imgui.Image(imgui.TextureID(win.scr.screenTexture),
		imgui.Vec2{
			win.scr.scaledWidth(),
			win.scr.scaledHeight(),
		})
	win.isHovered = imgui.IsItemHovered()

	imgui.PopStyleVarV(2)

	imgui.End()
}
