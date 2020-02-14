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

func setColors() {
	style := imgui.CurrentStyle()
	style.SetColor(imgui.StyleColorWindowBg, imgui.Vec4{0.075, 0.08, 0.09, 0.75})
	style.SetColor(imgui.StyleColorTitleBg, imgui.Vec4{0.075, 0.08, 0.09, 1.0})
	style.SetColor(imgui.StyleColorMenuBarBg, imgui.Vec4{0.075, 0.08, 0.09, 1.0})
}
