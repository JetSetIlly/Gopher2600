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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui/fonts"
)

// master control for enabling deeppoke capability.
const allowDeepPoke = true

// update will run deepPoke() or current() function depending on state of
// liveScope.
func (win *winTIA) update(deepPoke func(), current func()) {
	if !win.deepPoking {
		win.img.dbg.PushRawEvent(current)
		return
	}

	deepPoke()
}

func (win *winTIA) drawPersistenceControl() {
	win.scopeHeight = imguiMeasureHeight(func() {
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()

		if allowDeepPoke {
			if win.deepPoking {
				imgui.Text(fmt.Sprintf("%c Changes will be backtraced if possible and persist as appropriate", fonts.Persist))
			} else {
				imgui.Text(fmt.Sprintf("%c Changes will take effect going forward", fonts.GoingForward))
			}
			if imgui.IsItemClicked() {
				win.deepPoking = !win.deepPoking
			}
		} else {
			imgui.Text(fmt.Sprintf("%c Changes will take effect going forward", fonts.GoingForward))
		}
	})
}
