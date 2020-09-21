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
	"github.com/inkyblackness/imgui-go/v2"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
)

func (img *SdlImgui) drawPlusROMFirstInstallation() {
	if img.plusROMFirstInstallation == nil {
		return
	}

	nick := img.plusROMFirstInstallation.Cart.Prefs.Nick.String()
	id := img.plusROMFirstInstallation.Cart.Prefs.ID.String()

	img.hasModal = true

	imgui.OpenPopup("PlusROM First Installation")
	if imgui.BeginPopupModalV("PlusROM First Installation", nil, imgui.WindowFlagsAlwaysAutoResize) {
		imgui.Text("This looks like your first time using a PlayROM cartridge. Before")
		imgui.Text("proceeding it is a good idea for you to set your 'nick'. This will be")
		imgui.Text("used to identify you when contacting the PlayROM server.")

		imgui.Spacing()
		imgui.Spacing()

		imgui.AlignTextToFramePadding()
		imgui.Text("Nick")
		imgui.SameLine()

		if imguiTextInput("##nick", false, plusrom.MaxNickLength, &nick, true) {
			img.plusROMFirstInstallation.Cart.Prefs.Nick.Set(nick)
			img.plusROMFirstInstallation.Cart.Prefs.Save()
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
				img.plusROMFirstInstallation.Cart.Prefs.Nick.Set(nick)
				img.plusROMFirstInstallation.Cart.Prefs.Save()

				select {
				case img.plusROMFirstInstallation.Finish <- nil:
				default:
				}
				img.plusROMFirstInstallation = nil

				imgui.CloseCurrentPopup()
				img.hasModal = false
			}
		} else {
			imgui.AlignTextToFramePadding()
			imgui.Text("nick is not valid")
		}
	}
	imgui.EndPopup()
}
