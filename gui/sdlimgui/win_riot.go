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

const riotTitle = "RIOT"

type riot struct {
	img   *SdlImgui
	setup bool
}

func newRIOT(img *SdlImgui) (*riot, error) {
	riot := &riot{
		img: img,
	}

	return riot, nil
}

// draw is called by service loop
func (riot *riot) draw() {
	if riot.img.vcs != nil {
		if !riot.setup {
			imgui.SetNextWindowPos(imgui.Vec2{790, 610})
			imgui.SetNextWindowSize(imgui.Vec2{464, 48})
			riot.setup = true
		}

		imgui.BeginV(riotTitle, nil, 0)
		imgui.Text(riot.img.vcs.RIOT.String())
		imgui.End()
	}
}
