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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
)

const winPlusROMPrefsTitle = "PlusROM Preferences"
const menuPlusROMPrefsTitle = "Preferences"

type winPlusROMPrefs struct {
	windowManagement

	img *SdlImgui
}

func newWinPlusROMPrefs(img *SdlImgui) (managedWindow, error) {
	win := &winPlusROMPrefs{
		img: img,
	}

	return win, nil
}

func (win *winPlusROMPrefs) init() {
}

func (win *winPlusROMPrefs) destroy() {
}

func (win *winPlusROMPrefs) id() string {
	return winPlusROMPrefsTitle
}

func (win *winPlusROMPrefs) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.IsPlusROM {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{609, 55}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winPlusROMPrefsTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	nick := win.img.lz.Cart.PlusROMNick
	id := win.img.lz.Cart.PlusROMID

	imgui.AlignTextToFramePadding()
	imgui.Text("Nick")
	imgui.SameLine()

	if imguiTextInput("##nick", true, plusrom.MaxNickLength, &nick, false) {
		win.img.term.pushCommand(fmt.Sprintf("PLUSROM NICK %s", nick))
	}

	imgui.SameLine()
	imgui.AlignTextToFramePadding()
	imgui.Text("  ID")
	imgui.SameLine()
	imgui.Text(id)

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	imgui.Text("Nick and ID are set for ALL PlusROM cartridges")

	imgui.End()
}
