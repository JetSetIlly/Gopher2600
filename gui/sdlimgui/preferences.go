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

	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/resources"
)

// unified preferences for both modes (debugger and playmode). preferences
// should be reloaded whenever the emulation mode changes.
//
// in the case of most of the prefence values in this struct it won't matter
// because the preference value is either: the same for both modes, or only
// used as appropriate in other areas of the gui package.
//
// the one value that is tricky to handle is the audioEnabled flag. what we
// don't want is to check the emulation mode every time the audio buffer is
// updated. we solve that by registering a callback function which is run
// whenever the value is set (even if the value hasn't changed).
type preferences struct {
	img *SdlImgui

	// two disk objects so we can load and save the  preferences assigned to
	// them separately. both use the same prefs file.
	//
	// dsk is created during newPreferences() and dskWin is created during
	// setWindowPreferences()
	dsk    *prefs.Disk
	dskWin *prefs.Disk

	// debugger preferences
	openOnError  prefs.Bool
	audioEnabled prefs.Bool

	// playmode preferences
	controllerNotifcations    prefs.Bool
	plusromNotifications      prefs.Bool
	superchargerNotifications prefs.Bool

	// window preferences. these are attached to dskWin rather than dsk.
	fullScreen prefs.Bool
}

func newPreferences(img *SdlImgui) (*preferences, error) {
	p := &preferences{img: img}

	// defaults
	p.openOnError.Set(true)
	p.audioEnabled.Set(true)
	p.controllerNotifcations.Set(true)
	p.plusromNotifications.Set(true)
	p.superchargerNotifications.Set(true)

	// setup preferences
	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}

	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}

	// debugger
	err = p.dsk.Add("sdlimgui.debugger.terminalOnError", &p.openOnError)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("sdlimgui.debugger.audioEnabled", &p.audioEnabled)
	if err != nil {
		return nil, err
	}

	p.audioEnabled.RegisterCallback(func(enabled prefs.Value) error {
		if img.isPlaymode() {
			p.img.audio.Mute(false)
		} else {
			p.img.audio.Mute(!enabled.(bool))
		}
		return nil
	})

	// playmode
	err = p.dsk.Add("sdlimgui.playmode.controllerNotifcations", &p.controllerNotifcations)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("sdlimgui.playmode.plusromNotifcations", &p.plusromNotifications)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("sdlimgui.playmode.superchargerNotifications", &p.superchargerNotifications)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Load(true)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *preferences) setWindowPreferences(isPlayMode bool) error {
	// save existing windows preferences if necessary
	if p.dskWin != nil {
		err := p.dskWin.Save()
		if err != nil {
			return err
		}
	}

	var group string

	if isPlayMode {
		group = "sdlimgui.playmode"
	} else {
		group = "sdlimgui.debugger"
	}

	// setup preferences
	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return err
	}

	p.dskWin, err = prefs.NewDisk(pth)
	if err != nil {
		return err
	}

	p.fullScreen.RegisterCallback(func(v prefs.Value) error {
		p.img.plt.setFullScreen(p.fullScreen.Get().(bool))
		return nil
	})
	err = p.dskWin.Add(fmt.Sprintf("%s.fullscreen", group), &p.fullScreen)
	if err != nil {
		return err
	}

	err = p.dskWin.Add(fmt.Sprintf("%s.windowSize", group), prefs.NewGeneric(
		func(s string) error {
			var w, h int32
			_, err := fmt.Sscanf(s, "%d,%d", &w, &h)
			if err != nil {
				return err
			}
			p.img.plt.window.SetSize(w, h)
			return nil
		},
		func() string {
			w, h := p.img.plt.window.GetSize()
			return fmt.Sprintf("%d,%d", w, h)
		},
	))
	if err != nil {
		return err
	}

	err = p.dskWin.Add(fmt.Sprintf("%s.windowPos", group), prefs.NewGeneric(
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
			p.img.plt.window.SetPosition(x, y)
			return nil
		},
		func() string {
			x, y := p.img.plt.window.GetPosition()
			return fmt.Sprintf("%d,%d", x, y)
		},
	))
	if err != nil {
		return err
	}

	err = p.dskWin.Load(true)
	if err != nil {
		return err
	}

	return nil
}

// load preferences from disk.
func (p *preferences) load() error {
	return p.dsk.Load(false)
}

// save preferences to disk.
func (p *preferences) save() error {
	return p.dsk.Save()
}

// loadWin preferences from disk.
// func (p *preferences) loadWin() error {
// 	return p.dskWin.Load(false)
// }.

// saveWin preferences to disk.
func (p *preferences) saveWin() error {
	return p.dskWin.Save()
}
