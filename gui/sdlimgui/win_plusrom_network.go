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

	"github.com/inkyblackness/imgui-go/v2"
)

const winPlusROMNetworkTitle = "PlusROM Network"
const menuPlusROMNetworkTitle = "Network"

type winPlusROMNetwork struct {
	windowManagement

	img *SdlImgui
}

func newWinPlusROMNetwork(img *SdlImgui) (managedWindow, error) {
	win := &winPlusROMNetwork{
		img: img,
	}

	return win, nil
}

func (win *winPlusROMNetwork) init() {
}

func (win *winPlusROMNetwork) destroy() {
}

func (win *winPlusROMNetwork) id() string {
	return winPlusROMNetworkTitle
}

func (win *winPlusROMNetwork) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.IsPlusROM {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{659, 35}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winPlusROMNetworkTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	host := win.img.lz.Cart.PlusROMAddrInfo.Host
	path := win.img.lz.Cart.PlusROMAddrInfo.Path

	imgui.AlignTextToFramePadding()
	imgui.Text("Hostname")
	imgui.SameLine()
	if imgui.InputText("##hostname", &host) {
		win.img.term.pushCommand(fmt.Sprintf("PLUSROM HOST %s", host))
	}

	imgui.AlignTextToFramePadding()
	imgui.Text("    Path")
	imgui.SameLine()
	if imgui.InputText("##path", &path) {
		win.img.lz.Dbg.PushRawEvent(func() {
			win.img.term.pushCommand(fmt.Sprintf("PLUSROM PATH %s", path))
		})
	}

	imgui.End()
}
