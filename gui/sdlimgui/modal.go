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
	"github.com/jetsetilly/imgui-go/v5"
)

type modal int

const (
	modalNone modal = iota
	modalPlusROMFirstInstallation
	modalUnsupportedDWARF
	modalElfUndefinedSymbols
)

func (img *SdlImgui) modalDraw() {
	switch img.modal {
	case modalPlusROMFirstInstallation:
		img.modalDrawPlusROMFirstInstallation()
	case modalUnsupportedDWARF:
		img.modalDrawUnsupportedDWARF()
	case modalElfUndefinedSymbols:
		img.modalElfUndefinedSymbols()
	}
}

func (img *SdlImgui) modalActive() bool {
	return img.modal != modalNone
}

func (img *SdlImgui) modalElfUndefinedSymbols() {
	const popupTitle = "Undefined Symbols in ELF file"

	imgui.OpenPopup(popupTitle)
	flgs := imgui.WindowFlagsAlwaysAutoResize
	flgs |= imgui.WindowFlagsNoMove
	flgs |= imgui.WindowFlagsNoSavedSettings
	if imgui.BeginPopupModalV(popupTitle, nil, flgs) {
		imgui.Text("The ELF ROM contains an undefined symbol. Rather than aborting")
		imgui.Text("the loading process the symbol has been linked to the undefined")
		imgui.Text("symbol handler")
		imgui.Spacing()
		imgui.Spacing()
		imgui.Text("The ROM will progress as expected until the symbol is accessed")
		imgui.Spacing()
		imgui.Spacing()
		imgui.Text("If the symbol is accessed during execution a memory fault will")
		imgui.Text("be generated")

		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()

		sz := imgui.ContentRegionAvail()
		sz.Y = img.fonts.guiSize
		sz.Y += imgui.CurrentStyle().CellPadding().Y * 2
		sz.Y += imgui.CurrentStyle().FramePadding().Y * 2
		if imgui.ButtonV("Continue with 'undefined function' handler", sz) {
			imgui.CloseCurrentPopup()
			img.modal = modalNone
		}

		imgui.EndPopup()
	}
}
