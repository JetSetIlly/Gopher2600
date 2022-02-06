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

	"github.com/jetsetilly/gopher2600/emulation"
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

	// sdlimgui preferences on disk
	dsk *prefs.Disk

	// debugger preferences
	openOnError  prefs.Bool
	audioEnabled prefs.Bool

	// playmode preferences
	controllerNotifcations    prefs.Bool
	plusromNotifications      prefs.Bool
	superchargerNotifications prefs.Bool

	// fonts
	guiFont             prefs.Float
	codeFont            prefs.Float
	codeFontLineSpacing prefs.Int

	// window preferences are split over two prefs.Disk instances, to allow
	// geometry to be saved at a different time to the fullscreen preference
	dskWinGeom       *prefs.Disk
	dskWinFullScreen *prefs.Disk

	// full screen preference
	fullScreen prefs.Bool
}

func newPreferences(img *SdlImgui) (*preferences, error) {
	p := &preferences{img: img}

	// defaults
	p.openOnError.Set(true)
	p.audioEnabled.Set(false)
	p.controllerNotifcations.Set(true)
	p.plusromNotifications.Set(true)
	p.superchargerNotifications.Set(true)
	p.guiFont.Set(13.0)
	p.codeFont.Set(15.0)
	p.codeFontLineSpacing.Set(2.0)

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

	p.audioEnabled.SetHookPost(func(enabled prefs.Value) error {
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

	// fonts (only used when compiled with imguifreetype build tag)
	err = p.dsk.Add("sdlimgui.fonts.gui", &p.guiFont)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.fonts.code", &p.codeFont)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.fonts.codeLineSpacing", &p.codeFontLineSpacing)
	if err != nil {
		return nil, err
	}

	// load off disk
	err = p.dsk.Load(true)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// load preferences from disk. does not load window preferences.
func (p *preferences) load() error {
	return p.dsk.Load(false)
}

// save preferences to disk. does not save window preferences.
func (p *preferences) save() error {
	return p.dsk.Save()
}

// load window preferences for whatever mode we're currently in.
func (p *preferences) loadWindowPreferences() error {
	// save existing windows preferences if necessary
	err := p.saveWindowPreferences()
	if err != nil {
		return err
	}

	var group string

	switch p.img.mode {
	case emulation.ModeNone:
		p.img.plt.window.Hide()
	case emulation.ModeDebugger:
		group = "sdlimgui.debugger"
	case emulation.ModePlay:
		group = "sdlimgui.playmode"
	default:
		panic(fmt.Sprintf("cannot set window mode for unsupported emulation mode (%v)", p.img.mode))
	}

	// setup preferences
	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return err
	}

	// force window out of fullscreen. if we don't we can't guarantee that the
	// positioning of the window occurs before the full screen setting is
	// applied.
	//
	// this is noticeable when moving from an emulation mode with fullscreen
	// set to a mode with it unset. similar to how moving from a large window
	// to a small window
	p.img.plt.setFullScreen(false)

	// full screen preferences
	p.dskWinFullScreen, err = prefs.NewDisk(pth)
	if err != nil {
		return err
	}

	p.fullScreen.SetHookPre(func(v prefs.Value) error {
		// save window geometry if we're not *currently* in fullscreen mode
		// (this is a pre hook)
		//
		// a post hook is no good because it means the wrong geometry will be
		// saved. we want to save the non-fullscreen user preference.
		if !p.fullScreen.Get().(bool) {
			if p.dskWinGeom != nil {
				err := p.dskWinGeom.Save()
				if err != nil {
					return err
				}
			}
		}
		p.img.plt.setFullScreen(v.(bool))
		return nil
	})
	err = p.dskWinFullScreen.Add(fmt.Sprintf("%s.fullscreen", group), &p.fullScreen)
	if err != nil {
		return err
	}

	// window geometry preferences
	p.dskWinGeom, err = prefs.NewDisk(pth)
	if err != nil {
		return err
	}

	err = p.dskWinGeom.Add(fmt.Sprintf("%s.windowGeometry", group), prefs.NewGeneric(
		func(s string) error {
			var w, h int32
			var x, y int32
			_, err := fmt.Sscanf(s, "%d, %d, %d, %d", &x, &y, &w, &h)
			if err != nil {
				return err
			}

			// set size before position. if we don't then switching from a
			// larger window to a smaller window will not be positioned
			// correctly.
			p.img.plt.window.SetSize(w, h)
			p.img.plt.window.SetPosition(x, y)

			return nil
		},
		func() string {
			// if emulation is not running full screen, return the window
			// geometry...
			if !p.fullScreen.Get().(bool) {
				x, y := p.img.plt.window.GetPosition()
				w, h := p.img.plt.window.GetSize()
				return fmt.Sprintf("%d, %d, %d, %d", x, y, w, h)
			}

			// ... otherwise, indicate that the previous value is to be used
			return prefs.GenericGetValueUndefined
		},
	))
	if err != nil {
		return err
	}

	err = p.dskWinGeom.Load(true)
	if err != nil {
		return err
	}

	err = p.dskWinFullScreen.Load(true)
	if err != nil {
		return err
	}

	return nil
}

// save window preferences to disk. saves preferences for whatever emulation
// mode we're currently in.
func (p *preferences) saveWindowPreferences() error {
	if p.dskWinFullScreen != nil {
		err := p.dskWinFullScreen.Save()
		if err != nil {
			return err
		}
	}

	if p.dskWinGeom != nil {
		err := p.dskWinGeom.Save()
		if err != nil {
			return err
		}
	}

	return nil
}
