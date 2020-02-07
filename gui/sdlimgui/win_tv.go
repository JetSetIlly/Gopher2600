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

const tvTitle = "TV Status"

type tv struct {
	img   *SdlImgui
	setup bool
}

func newTV(img *SdlImgui) (*tv, error) {
	tv := &tv{
		img: img,
	}

	return tv, nil
}

// draw is called by service loop
func (tv *tv) draw() {
	if tv.img.vcs != nil {
		if !tv.setup {
			imgui.SetNextWindowPos(imgui.Vec2{685, 267})
			imgui.SetNextWindowSize(imgui.Vec2{211, 59})
			tv.setup = true
		}
		imgui.BeginV(tvTitle, nil, 0)
		imgui.Text(tv.img.vcs.TV.String())
		imgui.End()
	}
}
