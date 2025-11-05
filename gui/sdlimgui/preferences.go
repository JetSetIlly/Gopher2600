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

	"github.com/jetsetilly/gopher2600/debugger/govern"
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

	// prefs that will be saved automatically on program exit
	saveOnExitDsk *prefs.Disk

	// emulation preferences
	recentROM prefs.String

	// debugger preferences
	terminalOnError       prefs.Bool
	audioMuteDebugger     prefs.Bool
	showTooltips          prefs.Bool
	showTimelineThumbnail prefs.Bool
	disasmColour          prefs.Bool
	disasmSequential      prefs.Bool
	disasmGroupScanlines  prefs.Bool
	pxeColourIndicators   prefs.Bool

	// playmode preferences
	audioMutePlaymode    prefs.Bool
	fpsDetail            prefs.Bool
	activePause          prefs.Bool
	paddleOnMouseCapture prefs.Bool

	// playmode notifications
	controllerNotifcations    prefs.Bool
	plusromNotifications      prefs.Bool
	superchargerNotifications prefs.Bool
	audioMuteNotification     prefs.Bool
	notificationVisibility    prefs.Float
	memoryUsageInOverlay      prefs.Bool
	frameQueueMeterInOverlay  prefs.Bool
	audioQueueMeterInOverlay  prefs.Bool

	// fonts
	guiFontSize         prefs.Int
	terminalFontSize    prefs.Int
	codeFontSize        prefs.Int
	codeFontLineSpacing prefs.Int

	// display
	frameQueueLenAuto prefs.Bool
	frameQueueLen     prefs.Int
	glSwapInterval    prefs.Int

	// window preferences are split over two prefs.Disk instances, to allow
	// geometry to be saved at a different time to the fullscreen preference
	dskWinGeom       *prefs.Disk
	dskWinFullScreen *prefs.Disk

	// full screen preference. will be set according to the current emulation mode
	fullScreen prefs.Bool
}

func newPreferences(img *SdlImgui) (*preferences, error) {
	p := &preferences{img: img}

	// setup preferences
	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}

	// main disk preferences
	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("sdlimgui.emulation.recentrom", &p.recentROM)
	if err != nil {
		return nil, err
	}

	// debugger options
	err = p.dsk.Add("sdlimgui.debugger.terminalOnError", &p.terminalOnError)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.debugger.showTooltips", &p.showTooltips)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.debugger.showTimelineThumbnail", &p.showTimelineThumbnail)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.debugger.disasm.color", &p.disasmColour)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.debugger.disasm.sequential", &p.disasmSequential)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.debugger.disasm.groupScanlines", &p.disasmGroupScanlines)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.debugger.pxe.colorIndicators", &p.pxeColourIndicators)
	if err != nil {
		return nil, err
	}

	// debugger audio mute options later

	// playmode options
	err = p.dsk.Add("sdlimgui.playmode.fpsDetail", &p.fpsDetail)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.playmode.activePause", &p.activePause)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.playmode.paddleOnMouseCapture", &p.paddleOnMouseCapture)
	if err != nil {
		return nil, err
	}
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
	err = p.dsk.Add("sdlimgui.playmode.audioMuteNotification", &p.audioMuteNotification)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.playmode.notifcationVisibility", &p.notificationVisibility)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.playmode.memoryUsageInOverlay", &p.memoryUsageInOverlay)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.playmode.frameQueueMeterInOverlay", &p.frameQueueMeterInOverlay)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.playmode.audioQueueMeterInOverlay", &p.audioQueueMeterInOverlay)
	if err != nil {
		return nil, err
	}

	// playmode audio mute options later

	// fonts (only used when compiled with imguifreetype build tag)
	err = p.dsk.Add("sdlimgui.fonts.size.gui", &p.guiFontSize)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.fonts.size.terminal", &p.terminalFontSize)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.fonts.size.code", &p.codeFontSize)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("sdlimgui.fonts.codeLineSpacing", &p.codeFontLineSpacing)
	if err != nil {
		return nil, err
	}

	// display options

	err = p.dsk.Add("sdlimgui.display.frameQueueLen", &p.frameQueueLen)
	if err != nil {
		return nil, err
	}
	p.frameQueueLen.SetHookPost(func(v prefs.Value) error {
		p.img.screen.setFrameQueue(p.frameQueueLenAuto.Get().(bool), v.(int))
		return nil
	})

	err = p.dsk.Add("sdlimgui.display.frameQueueLenAuto", &p.frameQueueLenAuto)
	if err != nil {
		return nil, err
	}
	p.frameQueueLenAuto.SetHookPost(func(v prefs.Value) error {
		p.img.screen.setFrameQueue(v.(bool), p.frameQueueLen.Get().(int))
		return nil
	})

	err = p.dsk.Add("sdlimgui.display.glswapinterval", &p.glSwapInterval)
	if err != nil {
		return nil, err
	}
	p.glSwapInterval.SetConstraint(func(v prefs.Value) any {
		i := v.(int)
		if i > 1 {
			i = 1
		} else if i < -1 {
			i = -1
		}
		return i
	})
	p.glSwapInterval.SetHookPost(func(v prefs.Value) error {
		p.img.plt.setSwapInterval(v.(int))
		return nil
	})

	// audio mute options
	err = p.dsk.Add("sdlimgui.debugger.audioMute", &p.audioMuteDebugger)
	if err != nil {
		return nil, err
	}
	p.audioMuteDebugger.SetHookPost(func(muted prefs.Value) error {
		p.img.applyAudioMutePreference()
		return nil
	})

	err = p.dsk.Add("sdlimgui.playmode.audioMute", &p.audioMutePlaymode)
	if err != nil {
		return nil, err
	}
	p.audioMutePlaymode.SetHookPost(func(muted prefs.Value) error {
		p.img.applyAudioMutePreference()
		return nil
	})

	// disk preferences that will be saved on program exit
	p.saveOnExitDsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}
	err = p.saveOnExitDsk.Add("sdlimgui.emulation.recentrom", &p.recentROM)
	if err != nil {
		return nil, err
	}
	err = p.saveOnExitDsk.Add("sdlimgui.debugger.showTooltips", &p.showTooltips)
	if err != nil {
		return nil, err
	}
	err = p.saveOnExitDsk.Add("sdlimgui.debugger.disasm.color", &p.disasmColour)
	if err != nil {
		return nil, err
	}
	err = p.saveOnExitDsk.Add("sdlimgui.debugger.disasm.sequential", &p.disasmSequential)
	if err != nil {
		return nil, err
	}
	err = p.saveOnExitDsk.Add("sdlimgui.debugger.disasm.groupScanlines", &p.disasmGroupScanlines)
	if err != nil {
		return nil, err
	}
	err = p.saveOnExitDsk.Add("sdlimgui.debugger.pxe.colorIndicators", &p.pxeColourIndicators)
	if err != nil {
		return nil, err
	}
	err = p.saveOnExitDsk.Add("sdlimgui.playmode.fpsDetail", &p.fpsDetail)
	if err != nil {
		return nil, err
	}
	err = p.saveOnExitDsk.Add("sdlimgui.debugger.audioMute", &p.audioMuteDebugger)
	if err != nil {
		return nil, err
	}
	err = p.saveOnExitDsk.Add("sdlimgui.playmode.audioMute", &p.audioMutePlaymode)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *preferences) setDefaults() {
	// recentROM does not have a default value
	p.terminalOnError.Set(true)
	p.audioMuteDebugger.Set(true)
	p.showTooltips.Set(true)
	p.showTimelineThumbnail.Set(false)
	p.disasmColour.Set(true)
	p.disasmSequential.Set(false)
	p.disasmGroupScanlines.Set(false)
	p.pxeColourIndicators.Set(false)
	p.fpsDetail.Set(false)
	p.activePause.Set(false)
	p.paddleOnMouseCapture.Set(true)
	p.audioMutePlaymode.Set(false)
	p.controllerNotifcations.Set(true)
	p.plusromNotifications.Set(true)
	p.superchargerNotifications.Set(true)
	p.audioMuteNotification.Set(true)
	p.notificationVisibility.Set(0.75)
	p.memoryUsageInOverlay.Set(false)
	p.frameQueueMeterInOverlay.Set(false)
	p.audioQueueMeterInOverlay.Set(false)
	p.guiFontSize.Set(13)
	p.terminalFontSize.Set(12)
	p.codeFontSize.Set(15)
	p.codeFontLineSpacing.Set(2.0)
	p.frameQueueLenAuto.Set(false)
	p.frameQueueLen.Set(3)
	p.glSwapInterval.Set(1)
}

// load preferences from disk. does not load window preferences.
func (p *preferences) load() error {
	// calling set defaults before loading the values from disk. this makes sure
	// that the value hooks have been called at least once
	//
	// this is important because if the value is not on disk (eg. on first use
	// of the emulator) then the hook will not be triggered by the load process
	p.setDefaults()

	return p.dsk.Load()
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

	// hide window for duration of loading
	p.img.plt.window.Hide()
	defer func() {
		if p.img.mode.Load().(govern.Mode) != govern.ModeNone {
			p.img.plt.window.Show()
		}
	}()

	// force window out of fullscreen. if we don't we can't guarantee that the
	// positioning of the window occurs before the full screen setting is
	// applied.
	//
	// this is noticeable when moving from an emulation mode with fullscreen
	// set to a mode with it unset. similar to how moving from a large window
	// to a small window
	p.img.plt.setFullScreen(false)

	// define preferences group for the mode
	var group string

	switch p.img.mode.Load().(govern.Mode) {
	case govern.ModeDebugger:
		group = "sdlimgui.debugger"
	case govern.ModePlay:
		group = "sdlimgui.playmode"
	default:
		panic(fmt.Sprintf("cannot set window mode for unsupported emulation mode (%v)", p.img.mode))
	}

	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return err
	}

	// full screen preferences
	p.dskWinFullScreen, err = prefs.NewDisk(pth)
	if err != nil {
		return err
	}

	p.fullScreen.SetHookPre(func(v prefs.Value) error {
		// do nothing if value isn't changing
		if v.(bool) == p.fullScreen.Get().(bool) {
			return nil
		}

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
		func(s prefs.Value) error {
			var w, h int32
			var x, y int32
			_, err := fmt.Sscanf(s.(string), "%d, %d, %d, %d", &x, &y, &w, &h)
			if err != nil {
				return err
			}

			// set size before position. if we don't then switching from a
			// larger window to a smaller window will not be positioned
			// correctly.
			//
			// wrapping the resizing and repositioning in a Hide()/Show() pair.
			// without this the resizing is ineffective in some situations
			// related to whether the window has been "maximised" - distinct
			// from fullscreen mode
			//
			// (bug seen in X11 with the cinnamon desktop. might no be present
			// in other environments)
			p.img.plt.window.SetSize(w, h)
			p.img.plt.window.SetPosition(x, y)

			return nil
		},
		func() prefs.Value {
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

	err = p.dskWinGeom.Load()
	if err != nil {
		return err
	}

	err = p.dskWinFullScreen.Load()
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
