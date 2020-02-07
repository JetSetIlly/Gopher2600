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
	img   *SdlImgui
	setup bool
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
		if !delays.setup {
			imgui.SetNextWindowPos(imgui.Vec2{1027, 27})
			imgui.SetNextWindowSize(imgui.Vec2{201, 313})
			delays.setup = true
		}
		imgui.BeginV(delaysTitle, nil, 0)
		imgui.Text(delays.img.vcs.TIA.Delay.String())
		imgui.Text(delays.img.vcs.TIA.Video.Player0.Delay.String())
		imgui.Text(delays.img.vcs.TIA.Video.Player1.Delay.String())
		imgui.Text(delays.img.vcs.TIA.Video.Missile0.Delay.String())
		imgui.Text(delays.img.vcs.TIA.Video.Missile1.Delay.String())
		imgui.Text(delays.img.vcs.TIA.Video.Ball.Delay.String())
		imgui.End()
	}
}
