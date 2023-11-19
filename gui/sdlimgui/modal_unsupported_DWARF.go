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
	"github.com/jetsetilly/gopher2600/gui/fonts"
)

func (img *SdlImgui) modalDrawUnsupportedDWARF() {
	const popupTitle = "Unsupported DWARF Data"

	imgui.OpenPopup(popupTitle)
	flgs := imgui.WindowFlagsAlwaysAutoResize
	flgs |= imgui.WindowFlagsNoMove
	flgs |= imgui.WindowFlagsNoSavedSettings
	if imgui.BeginPopupModalV(popupTitle, nil, flgs) {
		imgui.BeginTable("##modalDWARFtable", 2)
		imgui.TableNextRow()

		imgui.TableNextColumn()
		imgui.PushFont(img.fonts.veryLargeFontAwesome)
		imgui.Text(string(fonts.Developer))
		imgui.PopFont()
		imgui.SameLine()
		imgui.Text(" ")

		imgui.TableNextColumn()
		imgui.Text("The DWARF data being loaded for your development purposes is")
		imgui.Text("not supported and will not be used")
		imgui.Text("")
		imgui.Text("If you're using the GNU compiler to preare the ROM, the debugging")
		imgui.Text("options should be:")
		imgui.Text("")
		imgui.PushFont(img.fonts.code)
		imgui.Text("    -g3 -gdwarf-4 -strict-dwarf")
		imgui.PopFont()
		imgui.Text("")
		imgui.Text("For other compilers it's important that the emitted DWARF data")
		imgui.Text("complies with version 4 with no custom operators")

		imgui.EndTable()

		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		sz := imgui.ContentRegionAvail()
		sz.Y = img.fonts.defaultFontSize
		sz.Y += imgui.CurrentStyle().CellPadding().Y * 2
		sz.Y += imgui.CurrentStyle().FramePadding().Y * 2
		if imgui.ButtonV("Continue without DWARF data", sz) {
			imgui.CloseCurrentPopup()
			img.modal = modalNone
		}

		imgui.EndPopup()
	}
}
