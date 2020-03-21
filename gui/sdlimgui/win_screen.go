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
	"fmt"

	"github.com/inkyblackness/imgui-go/v2"
)

const winScreenTitle = "TV Screen"

type winScreen struct {
	windowManagement
	img *SdlImgui
	scr *screen

	// is screen currently pointed at
	isHovered bool

	// the tv screen has captured mouse input
	isCaptured bool

	threeDigitDim imgui.Vec2
	fiveDigitDim  imgui.Vec2
}

func newWinScreen(img *SdlImgui) (managedWindow, error) {
	win := &winScreen{
		img: img,
		scr: img.screen,
	}

	return win, nil
}

func (win *winScreen) init() {
	win.threeDigitDim = imguiGetFrameDim("000")
	win.fiveDigitDim = imguiGetFrameDim("00000")
}

func (win *winScreen) destroy() {
}

func (win *winScreen) id() string {
	return winScreenTitle
}

// draw is called by service loop
func (win *winScreen) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{8, 28}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})

	// if isCaptured flag is set then change the title and border colors of the
	// TV Screen window.
	if win.isCaptured {
		imgui.PushStyleColor(imgui.StyleColorTitleBgActive, win.img.cols.CapturedScreenTitle)
		imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.CapturedScreenBorder)
	}

	imgui.BeginV(winScreenTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	// once the window has been drawn then remove any additional styling
	if win.isCaptured {
		imgui.PopStyleColorV(2)
	}

	imgui.Spacing()

	// actual display
	imgui.Image(imgui.TextureID(win.scr.screenTexture),
		imgui.Vec2{
			win.scr.scaledWidth(),
			win.scr.scaledHeight(),
		})
	win.isHovered = imgui.IsItemHovered()

	// tv status line
	imguiText("Frame:")
	imguiText(fmt.Sprintf("%-4d", win.img.lazy.TV.Frame))
	imgui.SameLineV(0, 15)
	imguiText("Scanline:")
	scanline := win.img.lazy.TV.Scanline
	imguiText(fmt.Sprintf("%-4d", scanline))
	imgui.SameLineV(0, 15)
	imguiText("Horiz Pos:")
	imguiText(fmt.Sprintf("%-4d", win.img.lazy.TV.HP))

	// fps indicator
	imgui.SameLineV(0, 20)
	imgui.AlignTextToFramePadding()
	if win.img.paused {
		imguiText("no fps")
	} else {
		if win.img.lazy.TV.ReqFPS < 1.0 {
			imguiText("< 1 fps")
		} else {
			imguiText(fmt.Sprintf("%03.1f fps", win.img.lazy.TV.AcutalFPS))
		}
	}

	// display toggles
	imgui.Spacing()
	imgui.Checkbox("Debug Colours", &win.scr.useAltPixels)
	imgui.SameLine()
	if imgui.Checkbox("Cropping", &win.scr.cropped) {
		win.scr.setCropping(win.scr.cropped)
	}

	imgui.End()
}
