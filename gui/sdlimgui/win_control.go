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

const controlTitle = "Control"

type control struct {
	windowManagement
	img *SdlImgui
}

func newControl(img *SdlImgui) (managedWindow, error) {
	con := &control{
		img: img,
	}
	return con, nil
}

func (con *control) destroy() {
}

func (con *control) id() string {
	return controlTitle
}

func (con *control) draw() {
	if !con.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{883, 35}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(controlTitle, &con.open, 0)

	if imgui.Button("Run") {
		con.img.issueTermCommand("RUN")
	}

	imgui.SameLine()
	if imgui.Button("Halt") {
		con.img.issueTermCommand("HALT")
	}

	imgui.Text("Step:")
	imgui.SameLine()
	if imgui.Button("Frame") {
		con.img.issueTermCommand("STEP FRAME")
	}
	imgui.SameLine()
	if imgui.Button("Scanline") {
		con.img.issueTermCommand("STEP SCANLINE")
	}

	imgui.End()
}
