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
	"image"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

const winComparisonID = "Comparison"

type winComparison struct {
	playmodeWin

	img *SdlImgui

	cmpTexture       texture
	diffTexture      texture
	audioIsDifferent bool

	// render channels are given to use by the main emulation through a GUI request
	render     chan *image.RGBA
	diffRender chan *image.RGBA
	audioDiff  chan bool
}

func newWinComparison(img *SdlImgui) (window, error) {
	win := &winComparison{
		img: img,
	}

	win.cmpTexture = img.rnd.addTexture(shaderColor, true, true, nil)
	win.diffTexture = img.rnd.addTexture(shaderColor, true, true, nil)

	return win, nil
}

func (win *winComparison) init() {
}

func (win winComparison) id() string {
	return winComparisonID
}

func (win *winComparison) playmodeSetOpen(open bool) {
	win.playmodeWin.playmodeSetOpen(open)
}

func (win *winComparison) playmodeDraw() bool {
	if win.render == nil || win.diffRender == nil || win.audioDiff == nil {
		win.playmodeWin.playmodeSetOpen(false)
		return false
	}

	// receive new thumbnail data and copy to texture
	select {
	case image := <-win.render:
		if image != nil {
			win.cmpTexture.markForCreation()
			win.cmpTexture.render(image)
		}
	default:
	}

	// receive new thumbnail data and copy to texture
	select {
	case image := <-win.diffRender:
		if image != nil {
			win.diffTexture.markForCreation()
			win.diffTexture.render(image)
		}
	default:
	}

	// receive new thumbnail data and copy to texture
	select {
	case win.audioIsDifferent = <-win.audioDiff:
	default:
	}

	if !win.playmodeOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 75, Y: 75}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})

	if imgui.BeginV(win.playmodeID(win.id()), &win.playmodeOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.playmodeGeom.update()
	imgui.End()

	return true
}

func (win *winComparison) draw() {
	sz := imgui.Vec2{X: specification.WidthTV, Y: specification.HeightTV}.Times(2.5)
	imgui.Image(imgui.TextureID(win.cmpTexture.getID()), sz)
	imgui.Image(imgui.TextureID(win.diffTexture.getID()), sz)
	if win.audioIsDifferent {
		imguiColorLabelSimple("Audio is different", win.img.cols.False)
	} else {
		imguiColorLabelSimple("Audio is the same", win.img.cols.True)
	}
}
