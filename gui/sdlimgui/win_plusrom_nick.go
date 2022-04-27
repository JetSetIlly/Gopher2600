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
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom/plusnet"
	"github.com/jetsetilly/gopher2600/logger"
)

const winPlusROMNickID = "PlusROM Nick"
const winPlusROMNickMenu = "Nick"

type winPlusROMNick struct {
	debuggerWin

	img *SdlImgui
}

func newWinPlusROMNick(img *SdlImgui) (window, error) {
	win := &winPlusROMNick{
		img: img,
	}

	return win, nil
}

func (win *winPlusROMNick) init() {
}

func (win *winPlusROMNick) id() string {
	return winPlusROMNickID
}

func (win *winPlusROMNick) debuggerDraw() {
	if !win.debuggerOpen {
		return
	}

	if !win.img.lz.Cart.IsPlusROM {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{659, 35}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	imgui.End()
}

func (win *winPlusROMNick) draw() {
	// if drawPlusROMNick has returned true then save the change to disk immediately
	if drawPlusROMNick(win.img) {
		err := win.img.vcs.Instance.Prefs.PlusROM.Save()
		if err != nil {
			logger.Logf("sdlimgui", "could not save preferences: %v", err)
		}
	}
}

// draws nick text input and ID information. used by PlusROM Nick window and
// also the PlusROM tab in the main prefs window.
//
// returns true if nick has been changed. this indicates that the nick has been
// changed (or at least the textinput has been edited) but the change has not
// been saveed to the disk.
func drawPlusROMNick(img *SdlImgui) bool {
	var changed bool

	nick := img.vcs.Instance.Prefs.PlusROM.Nick.Get().(string)
	id := img.vcs.Instance.Prefs.PlusROM.ID.Get().(string)

	imgui.AlignTextToFramePadding()
	imgui.Text("Nick")
	imgui.SameLine()

	if imguiTextInput("##nick", plusnet.MaxNickLength, &nick, false) {
		err := img.vcs.Instance.Prefs.PlusROM.Nick.Set(nick)
		if err != nil {
			logger.Logf("sdlimgui", "could not set plusrom nick: %v", err)
		}
		changed = true
	}

	imgui.AlignTextToFramePadding()
	imgui.Text("  ID")
	imgui.SameLine()
	imgui.Text(id)

	imgui.Spacing()
	imgui.Text("Change your PlusROM nick with care")

	return changed
}
