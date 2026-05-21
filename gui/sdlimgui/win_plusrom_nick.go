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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom/plusnet"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/imgui-go/v5"
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

func (win *winPlusROMNick) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	// do not open window if this is not a plusrom cartridge
	_, ok := win.img.cache.VCS.Mem.Cart.GetContainer().(*plusrom.PlusROM)
	if !ok {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 659, Y: 35}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winPlusROMNick) draw() {
	// if drawPlusROMNick has returned true then save the change to disk immediately
	if drawPlusROMNick(win.img) {
		err := win.img.dbg.VCS().Env.Prefs.Cartridge.PlusROM.Save()
		if err != nil {
			logger.Logf(logger.Allow, "sdlimgui", "could not save preferences: %v", err)
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

	nick := img.dbg.VCS().Env.Prefs.Cartridge.PlusROM.Nick.Get().(string)
	id := img.dbg.VCS().Env.Prefs.Cartridge.PlusROM.ID.Get().(string)

	imgui.AlignTextToFramePadding()
	imgui.Text("Nick")
	imgui.SameLine()

	if imguiTextInput("##nick", plusnet.MaxNickLength, &nick, false) {
		err := img.dbg.VCS().Env.Prefs.Cartridge.PlusROM.Nick.Set(nick)
		if err != nil {
			logger.Logf(logger.Allow, "sdlimgui", "could not set plusrom nick: %v", err)
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
