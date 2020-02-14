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

package sdlimgui

import (
	"github.com/inkyblackness/imgui-go/v2"
)

const delaysTitle = "TIA Delays"

type delays struct {
	img *SdlImgui
}

func newDelays(img *SdlImgui) (*delays, error) {
	delays := &delays{
		img: img,
	}

	return delays, nil
}

// draw is called by service loop
func (delays *delays) draw() {
	if delays.img.vcs != nil {
		imgui.SetNextWindowPosV(imgui.Vec2{72, 476}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
		imgui.SetNextWindowSizeV(imgui.Vec2{240, 189}, imgui.ConditionFirstUseEver)
		imgui.BeginV(delaysTitle, nil, 0)

		s := delays.img.vcs.TIA.Delay.String()
		if len(s) > 0 {
			imgui.Text(s)
		}
		s = delays.img.vcs.TIA.Video.Player0.Delay.String()
		if len(s) > 0 {
			imgui.Text(s)
		}
		s = delays.img.vcs.TIA.Video.Player1.Delay.String()
		if len(s) > 0 {
			imgui.Text(s)
		}
		s = delays.img.vcs.TIA.Video.Missile0.Delay.String()
		if len(s) > 0 {
			imgui.Text(s)
		}
		s = delays.img.vcs.TIA.Video.Missile1.Delay.String()
		if len(s) > 0 {
			imgui.Text(s)
		}
		s = delays.img.vcs.TIA.Video.Ball.Delay.String()
		if len(s) > 0 {
			imgui.Text(s)
		}

		imgui.End()
	}
}
