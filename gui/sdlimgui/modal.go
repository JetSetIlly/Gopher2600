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

type modal int

const (
	modalNone modal = iota
	modalPlusROMFirstInstallation
	modalUnsupportedDWARF
)

func (img *SdlImgui) modalDraw() {
	switch img.modal {
	case modalPlusROMFirstInstallation:
		img.modalDrawPlusROMFirstInstallation()
	case modalUnsupportedDWARF:
		img.modalDrawUnsupportedDWARF()
	}
}

func (img *SdlImgui) modalActive() bool {
	return img.modal != modalNone
}
