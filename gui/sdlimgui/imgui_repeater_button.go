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
	"sync/atomic"
	"time"

	"github.com/jetsetilly/imgui-go/v5"
)

// repeaterButton is intended to be embedded in window types
type repeaterButton struct {
	img              *SdlImgui
	repeatID         string
	repeatTime       time.Time
	repeatFPSLimiter atomic.Value // bool
}

func (win *repeaterButton) repeatButton(id string, f func()) {
	win.repeatButtonV(id, f, imgui.Vec2{})
}

func (win *repeaterButton) repeatButtonV(id string, f func(), fill imgui.Vec2) {
	imgui.ButtonV(id, fill)
	if imgui.IsItemActive() {
		if id != win.repeatID {
			win.img.dbg.PushFunction(func() {
				v := win.img.dbg.VCS().TV.SetFPSLimit(false)
				win.repeatFPSLimiter.Store(v)
			})
			win.repeatID = id
			win.repeatTime = time.Now()
			f()
			return
		}

		dur := time.Since(win.repeatTime)
		if dur > 5e+8 { // half a second in nanoseconds
			f()
		}
	} else if imgui.IsItemDeactivated() {
		win.repeatID = ""
		win.img.dbg.PushFunction(func() {
			v := win.repeatFPSLimiter.Load().(bool)
			win.img.dbg.VCS().TV.SetFPSLimit(v)
		})
	}
}
