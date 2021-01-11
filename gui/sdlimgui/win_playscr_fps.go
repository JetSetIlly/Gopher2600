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
	"time"

	"github.com/inkyblackness/imgui-go/v3"
)

const winPlayScrFPSTitle = "FPS"

type winPlayScrFPS struct {
	img  *SdlImgui
	open bool

	pulse *time.Ticker

	// fps value being displayed
	fps string
}

// number of seconds between updates.
const fpsUpdateFreq = 1

func newWinPlayScrFPS(img *SdlImgui) window {
	win := &winPlayScrFPS{
		img:   img,
		pulse: time.NewTicker(time.Second * fpsUpdateFreq),
	}
	return win
}

func (win *winPlayScrFPS) init() {
}

func (win *winPlayScrFPS) destroy() {
}

func (win *winPlayScrFPS) id() string {
	return winPlayScrFPSTitle
}

func (win *winPlayScrFPS) menuLabel() string {
	return winPlayScrFPSTitle
}

func (win *winPlayScrFPS) isOpen() bool {
	return win.open
}

func (win *winPlayScrFPS) setOpen(open bool) {
	win.open = open
}

func (win *winPlayScrFPS) draw() {
	if !win.open || !win.img.playmode.Load().(bool) {
		return
	}

	win.updateFPS()

	imgui.SetNextWindowPos(imgui.Vec2{0, 0})

	imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.Transparent)

	imgui.BeginV(winLogTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize|
		imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|imgui.WindowFlagsNoDecoration)

	imgui.Text(win.fps)

	imgui.PopStyleColorV(2)
	imgui.End()
}

func (win *winPlayScrFPS) updateFPS() {
	select {
	case <-win.pulse.C:
	default:
		return
	}

	win.fps = fmt.Sprintf("%03.1f fps", win.img.tv.GetActualFPS())
}
