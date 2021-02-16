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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
)

const winPlusROMPrefsID = "PlusROM Preferences"
const winPlusROMPrefsMenu = "Preferences"

type winPlusROMPrefs struct {
	img  *SdlImgui
	open bool
}

func newWinPlusROMPrefs(img *SdlImgui) (window, error) {
	win := &winPlusROMPrefs{
		img: img,
	}

	return win, nil
}

func (win *winPlusROMPrefs) init() {
}

func (win *winPlusROMPrefs) id() string {
	return winPlusROMPrefsID
}

func (win *winPlusROMPrefs) isOpen() bool {
	return win.open
}

func (win *winPlusROMPrefs) setOpen(open bool) {
	win.open = open
}

func (win *winPlusROMPrefs) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.IsPlusROM {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{609, 55}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)

	nick := win.img.lz.Cart.PlusROMNick
	id := win.img.lz.Cart.PlusROMID

	imgui.AlignTextToFramePadding()
	imgui.Text("Nick")
	imgui.SameLine()

	if imguiTextInput("##nick", plusrom.MaxNickLength, &nick, false) {
		win.img.term.pushCommand(fmt.Sprintf("PLUSROM NICK %s", nick))
	}

	imgui.SameLine()
	imgui.AlignTextToFramePadding()
	imgui.Text("  ID")
	imgui.SameLine()
	imgui.Text(id)

	imguiSeparator()

	imgui.Text("Nick and ID are set for ALL PlusROM cartridges")

	imgui.End()
}
