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

func badgePentagonHollow(col imgui.Vec4) {
	p := imgui.CursorScreenPos()
	dl := imgui.WindowDrawList()
	radius := imgui.FrameHeight() / 5
	p.Y += imgui.TextLineHeight() * 0.75
	p.X -= 5

	dl.AddCircleV(p, radius, colorConvertFloat4ToU32(col), 5, 3)
}

func badgePentagon(col imgui.Vec4) {
	p := imgui.CursorScreenPos()
	dl := imgui.WindowDrawList()
	radius := 2 + imgui.FrameHeight()/5
	p.Y += imgui.TextLineHeight() * 0.75
	p.X -= 5

	dl.AddCircleFilledV(p, radius, colorConvertFloat4ToU32(col), 5)
}
