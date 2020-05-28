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

const winCollisionsTitle = "Collisions"

type winCollisions struct {
	windowManagement
	widgetDimensions

	img *SdlImgui

	// ready flag colors
	colFlgReadyOn  imgui.PackedColor
	colFlgReadyOff imgui.PackedColor
}

func newWinCollisions(img *SdlImgui) (managedWindow, error) {
	win := &winCollisions{
		img: img,
	}

	return win, nil
}

func (win *winCollisions) init() {
	win.widgetDimensions.init()
	win.colFlgReadyOn = imgui.PackedColorFromVec4(win.img.cols.CPUFlgRdyOn)
	win.colFlgReadyOff = imgui.PackedColorFromVec4(win.img.cols.CPUFlgRdyOff)
}

func (win *winCollisions) destroy() {
}

func (win *winCollisions) id() string {
	return winCollisionsTitle
}

func (win *winCollisions) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{659, 35}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winCollisionsTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	imgui.Text("CXM0P ")
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("%08b", win.img.lz.Collisions.CXM0P))

	imgui.Text("CXM1P ")
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("%08b", win.img.lz.Collisions.CXM1P))

	imgui.Text("CXP0FB")
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("%08b", win.img.lz.Collisions.CXP0FB))

	imgui.Text("CXP1FB")
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("%08b", win.img.lz.Collisions.CXP1FB))

	imgui.Text("CXM0FB")
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("%08b", win.img.lz.Collisions.CXM0FB))

	imgui.Text("CXM1FB")
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("%08b", win.img.lz.Collisions.CXM1FB))

	imgui.Text("CXBLPF")
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("%08b", win.img.lz.Collisions.CXBLPF))

	imgui.Text("CXPPMM")
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("%08b", win.img.lz.Collisions.CXPPMM))

	imgui.End()
}
