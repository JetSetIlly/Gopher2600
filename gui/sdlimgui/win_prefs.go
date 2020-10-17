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
	"github.com/jetsetilly/gopher2600/logger"
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
		win.img.term.pushCommand("PREFS TOGGLE RANDSTART")
	}

	if imgui.Checkbox("Random Pins", &win.img.lz.Prefs.RandomPins) {
		win.img.term.pushCommand("PREFS TOGGLE RANDPINS")
	}

	if imgui.Checkbox("Use Fxxx Mirror", &win.img.lz.Prefs.FxxxMirror) {
		win.img.term.pushCommand("PREFS TOGGLE FXXXMIRROR")
	}

	if imgui.Checkbox("Use Symbols", &win.img.lz.Prefs.Symbols) {
		win.img.term.pushCommand("PREFS TOGGLE SYMBOLS")
	}

	termOnError := win.img.wm.term.openOnError.Get().(bool)
	if imgui.Checkbox("Open Terminal on Error", &termOnError) {
		err := win.img.wm.term.openOnError.Set(termOnError)
		if err != nil {
			logger.Log("sdlimgui", fmt.Sprintf("could not set preference value: %v", err))
		}
	}

	if imgui.Button("Save") {
		err := win.img.prefs.Save()
		if err != nil {
			logger.Log("sdlimgui", fmt.Sprintf("could not save preferences: %v", err))
		}
		win.img.term.pushCommand("PREFS SAVE")
	}

	imgui.SameLine()
	if imgui.Button("Restore") {
		err := win.img.prefs.Load(false)
		if err != nil {
			logger.Log("sdlimgui", fmt.Sprintf("could not restore preferences: %v", err))
		}
		win.img.term.pushCommand("PREFS LOAD")
	}

	imgui.End()
}
