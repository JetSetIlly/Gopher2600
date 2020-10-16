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

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/paths"
	"github.com/jetsetilly/gopher2600/prefs"
)

// the acceptable preferencegroups provided to initPrefs().
type prefGroup string

const (
	prefsGrpDebugger prefGroup = "sdlimgui.debugger"
	prefsGrpPlaymode prefGroup = "sdlimgui.playmode"
)

// preferences change subtly when switching between debugger and play modes.
func (img *SdlImgui) initPrefs(group prefGroup) error {
	// setup preferences
	pth, err := paths.ResourcePath("", prefs.DefaultPrefsFile)
	if err != nil {
		return err
	}
	img.prefs, err = prefs.NewDisk(pth)
	if err != nil {
		return err
	}

	err = img.prefs.Add(fmt.Sprintf("%s.windowsize", group), prefs.NewGeneric(
		func(s string) error {
			var w, h int32
			_, err := fmt.Sscanf(s, "%d,%d", &w, &h)
			if err != nil {
				return err
			}
			img.plt.window.SetSize(w, h)
			return nil
		},
		func() string {
			w, h := img.plt.window.GetSize()
			return fmt.Sprintf("%d,%d", w, h)
		},
	))
	if err != nil {
		return err
	}

	err = img.prefs.Add(fmt.Sprintf("%s.windowpos", group), prefs.NewGeneric(
		func(s string) error {
			var x, y int32
			_, err := fmt.Sscanf(s, "%d,%d", &x, &y)
			if err != nil {
				return err
			}
			// !!TODO: SetPosition doesn't seem to set window position as you
			// might expect. On XWindow with Cinnamon WM, it seems to place the
			// window top to the window further down and slightly to the right
			// of where it should be. This means that the window "drifts" down
			// the screen on subsequent loads
			img.plt.window.SetPosition(x, y)
			return nil
		},
		func() string {
			x, y := img.plt.window.GetPosition()
			return fmt.Sprintf("%d,%d", x, y)
		},
	))
	if err != nil {
		return err
	}

	err = img.prefs.Add(fmt.Sprintf("%s.terminalOnError", group), &img.wm.term.openOnError)
	if err != nil {
		return err
	}

	// load preferences from disk
	err = img.prefs.Load(true)
	if err != nil {
		// ignore missing prefs file errors
		if !curated.Is(err, prefs.NoPrefsFile) {
			return err
		}
	}

	return nil
}
