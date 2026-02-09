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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom/plusnet"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/imgui-go/v5"
)

func (img *SdlImgui) modalDrawPlusROMFirstInstallation() {
	nick := img.dbg.VCS().Env.Prefs.Cartridge.PlusROM.Nick.String()
	id := img.dbg.VCS().Env.Prefs.Cartridge.PlusROM.ID.String()

	const popupTitle = "PlusROM First Installation"

	imgui.OpenPopup(popupTitle)
	flgs := imgui.WindowFlagsAlwaysAutoResize
	flgs |= imgui.WindowFlagsNoMove
	flgs |= imgui.WindowFlagsNoSavedSettings
	if imgui.BeginPopupModalV(popupTitle, nil, flgs) {
		imgui.Text("This looks like your first time using a PlusROM cartridge. Before")
		imgui.Text("proceeding it is a good idea for you to set your 'nick'. This will be")
		imgui.Text("used to identify you when contacting the PlusROM server.")

		imgui.Spacing()
		imgui.Spacing()

		imgui.AlignTextToFramePadding()
		imgui.Text("Nick")
		imgui.SameLine()

		if imguiTextInput("##nick", plusnet.MaxNickLength, &nick, true) {
			err := img.dbg.VCS().Env.Prefs.Cartridge.PlusROM.Nick.Set(nick)
			if err != nil {
				logger.Logf(logger.Allow, "sdlimgui", "could not set plusrom nick: %v", err)
			}
			err = img.dbg.VCS().Env.Prefs.Cartridge.PlusROM.Save()
			if err != nil {
				logger.Logf(logger.Allow, "sdlimgui", "could not save preferences: %v", err)
			}
		}

		imgui.SameLine()
		imgui.AlignTextToFramePadding()
		imgui.Text("  ID")
		imgui.SameLine()
		imgui.Text(id)

		imgui.Spacing()
		imgui.Spacing()

		if len(nick) >= 1 {
			if imgui.Button("I'm happy with my nick") {
				err := img.dbg.VCS().Env.Prefs.Cartridge.PlusROM.Nick.Set(nick)
				if err != nil {
					logger.Logf(logger.Allow, "sdlimgui", "could not set preference value: %v", err)
				}
				err = img.dbg.VCS().Env.Prefs.Cartridge.PlusROM.Save()
				if err != nil {
					logger.Logf(logger.Allow, "sdlimgui", "could not save preferences: %v", err)
				}

				imgui.CloseCurrentPopup()
				img.modal = modalNone
			}
		} else {
			imgui.AlignTextToFramePadding()
			imgui.Text("nick is not valid")
		}
	}
	imgui.EndPopup()
}
