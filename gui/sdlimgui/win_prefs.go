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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlimgui

import (
	"github.com/inkyblackness/imgui-go/v2"
)

const winPrefsTile = "Preferences"

type winPrefs struct {
	windowManagement
	img *SdlImgui
}

func newWinPrefs(img *SdlImgui) (managedWindow, error) {
	win := &winPrefs{
		img: img,
	}

	return win, nil
}

func (win *winPrefs) init() {
}

func (win *winPrefs) destroy() {
}

func (win *winPrefs) id() string {
	return winPrefsTile
}

func (win *winPrefs) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{10, 10}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winPrefsTile, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	if imgui.Checkbox("Random State (on startup)", &win.img.lz.Prefs.RandomState) {
		win.img.term.pushCommand("PREF TOGGLE RANDSTART")
	}

	if imgui.Checkbox("Random Pins", &win.img.lz.Prefs.RandomPins) {
		win.img.term.pushCommand("PREF TOGGLE RANDPINS")
	}

	if imgui.Button("Save") {
		win.img.term.pushCommand("PREF SAVE")
	}

	imgui.SameLine()
	if imgui.Button("Restore") {
		win.img.term.pushCommand("PREF LOAD")
	}

	imgui.End()
}
