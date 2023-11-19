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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/notifications"
)

const (
	notificationDurationPeripheral = 90
	notificationDurationCartridge  = 60
	notificationDurationEventRun   = 60
	notificationDurationScreenshot = 60
	notificationDurationEvent      = 10
)

// peripheralNotification is used to draw an indicator on the screen for controller change events.
type peripheralNotification struct {
	frames     int
	icon       string
	rightAlign bool
}

func (ntfy *peripheralNotification) set(peripheral plugging.PeripheralID) {
	ntfy.frames = notificationDurationPeripheral
	switch peripheral {
	case plugging.PeriphStick:
		ntfy.icon = fmt.Sprintf("%c", fonts.Stick)
	case plugging.PeriphPaddles:
		ntfy.icon = fmt.Sprintf("%c", fonts.Paddle)
	case plugging.PeriphKeypad:
		ntfy.icon = fmt.Sprintf("%c", fonts.Keypad)
	case plugging.PeriphSavekey:
		ntfy.icon = fmt.Sprintf("%c", fonts.Savekey)
	case plugging.PeriphGamepad:
		ntfy.icon = fmt.Sprintf("%c", fonts.Gamepad)
	case plugging.PeriphAtariVox:
		ntfy.icon = fmt.Sprintf("%c", fonts.AtariVox)
	default:
		ntfy.icon = ""
		return
	}
}

// pos should be the coordinate of the *extreme* bottom left or bottom right of
// the playscr window. the values will be adjusted according to whether we're
// display an icon or text.
func (ntfy *peripheralNotification) draw(win *playScr) {
	if ntfy.frames <= 0 {
		return
	}
	ntfy.frames--

	if !win.img.prefs.controllerNotifcations.Get().(bool) {
		return
	}

	// position window so that it is fully visible at the bottom of the screen.
	// taking special care of the right aligned window
	var id string
	var pos imgui.Vec2
	dimen := win.img.plt.displaySize()
	if ntfy.rightAlign {
		pos = imgui.Vec2{dimen[0], dimen[1]}
		id = "##rightPeriphNotification"
		pos.X -= win.img.fonts.gopher2600IconsSize * 1.35
	} else {
		pos = imgui.Vec2{0, dimen[1]}
		id = "##leftPeriphNotification"
		pos.X += win.img.fonts.gopher2600IconsSize * 0.20
	}
	pos.Y -= win.img.fonts.gopher2600IconsSize * 1.35

	imgui.SetNextWindowPos(pos)
	imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.Transparent)
	defer imgui.PopStyleColorV(2)

	imgui.PushFont(win.img.fonts.gopher2600Icons)
	defer imgui.PopFont()

	a := float32(win.img.prefs.notificationVisibility.Get().(float64))
	imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{a, a, a, a})
	defer imgui.PopStyleColor()

	periphOpen := true
	imgui.BeginV(id, &periphOpen, imgui.WindowFlagsAlwaysAutoResize|
		imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|
		imgui.WindowFlagsNoDecoration|imgui.WindowFlagsNoSavedSettings)

	imgui.Text(ntfy.icon)

	imgui.End()
}

// emulationEventNotification is used to draw an indicator on the screen for
// events defined in the emulation package.
type emulationEventNotification struct {
	open   bool
	frames int

	event notifications.Notify
	mute  bool
}

func (ntfy *emulationEventNotification) set(event notifications.Notify) {
	switch event {
	case notifications.NotifyRun:
		ntfy.event = event
		ntfy.frames = notificationDurationEventRun
		ntfy.open = true
	case notifications.NotifyPause:
		ntfy.event = event
		ntfy.frames = 0
		ntfy.open = true
	case notifications.NotifyScreenshot:
		ntfy.event = event
		ntfy.frames = notificationDurationScreenshot
		ntfy.open = true
	case notifications.NotifyMute:
		ntfy.event = event
		ntfy.frames = 0
		ntfy.open = true
		ntfy.mute = true
	case notifications.NotifyUnmute:
		ntfy.event = event
		ntfy.frames = 0
		ntfy.open = true
		ntfy.mute = false
	}
}

func (ntfy *emulationEventNotification) tick() {
	if ntfy.frames <= 0 {
		return
	}
	ntfy.frames--

	if ntfy.frames == 0 {
		if ntfy.mute {
			ntfy.event = notifications.NotifyMute
			ntfy.open = true
		} else {
			ntfy.open = false
		}
	}
}

func (ntfy *emulationEventNotification) draw(win *playScr, hosted bool) {
	ntfy.tick()
	if !ntfy.open {
		return
	}

	if !hosted {
		imgui.SetNextWindowPos(imgui.Vec2{X: 10, Y: 10})
		imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.Transparent)
		imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.Transparent)
		defer imgui.PopStyleColorV(2)

		a := float32(win.img.prefs.notificationVisibility.Get().(float64))
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{a, a, a, a})
		defer imgui.PopStyleColor()

		imgui.BeginV("##emulationNotification", &ntfy.open, imgui.WindowFlagsAlwaysAutoResize|
			imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|
			imgui.WindowFlagsNoDecoration|imgui.WindowFlagsNoSavedSettings|
			imgui.WindowFlagsNoBringToFrontOnFocus)
		defer imgui.End()

		imgui.PushFont(win.img.fonts.veryLargeFontAwesome)
		defer imgui.PopFont()
	}

	switch ntfy.event {
	case notifications.NotifyInitialising:
		imgui.Text("")
	case notifications.NotifyPause:
		imgui.Text(string(fonts.EmulationPause))
	case notifications.NotifyRun:
		imgui.Text(string(fonts.EmulationRun))
	case notifications.NotifyRewindBack:
		imgui.Text(string(fonts.EmulationRewindBack))
	case notifications.NotifyRewindFoward:
		imgui.Text(string(fonts.EmulationRewindForward))
	case notifications.NotifyRewindAtStart:
		imgui.Text(string(fonts.EmulationRewindAtStart))
	case notifications.NotifyRewindAtEnd:
		imgui.Text(string(fonts.EmulationRewindAtEnd))
	case notifications.NotifyScreenshot:
		imgui.Text(string(fonts.Camera))
	default:
		if ntfy.mute && win.img.prefs.audioMuteNotification.Get().(bool) {
			imgui.Text(string(fonts.AudioMute))
		}
	}
}

// cartridgeEventNotification is used to draw an indicator on the screen for cartridge
// events defined in the mapper package.
type cartridgeEventNotification struct {
	open   bool
	frames int

	event     notifications.Notify
	coprocDev bool
}

func (ntfy *cartridgeEventNotification) set(event notifications.Notify) {
	switch event {
	case notifications.NotifySuperchargerSoundloadStarted:
		ntfy.event = event
		ntfy.frames = 0
		ntfy.open = true
	case notifications.NotifySuperchargerSoundloadEnded:
		ntfy.event = event
		ntfy.frames = notificationDurationCartridge
		ntfy.open = true
	case notifications.NotifySuperchargerSoundloadRewind:
		ntfy.event = event
		ntfy.frames = notificationDurationCartridge
		ntfy.open = true
	case notifications.NotifyPlusROMNetwork:
		ntfy.event = event
		ntfy.frames = notificationDurationCartridge
		ntfy.open = true
	case notifications.NotifyCoprocDevStarted:
		ntfy.coprocDev = true
		ntfy.frames = 0
		ntfy.open = true
	case notifications.NotifyCoprocDevEnded:
		ntfy.coprocDev = false
		ntfy.frames = 0
		ntfy.open = false
	}
}

func (ntfy *cartridgeEventNotification) tick() {
	if ntfy.frames <= 0 {
		return
	}
	ntfy.frames--

	if ntfy.frames == 0 {
		// always remain open if coprocessor development is active
		if ntfy.coprocDev {
			ntfy.open = true
		} else {
			switch ntfy.event {
			case notifications.NotifySuperchargerSoundloadRewind:
				ntfy.event = notifications.NotifySuperchargerSoundloadStarted
			default:
				ntfy.open = false
			}
		}
	}
}

func (ntfy *cartridgeEventNotification) draw(win *playScr) {
	ntfy.tick()
	if !ntfy.open {
		return
	}

	// notifications are made up of an icon and a sub-icon. icons must be from
	// the gopher2600Icons font and the sub-icon from the largeFontAwesome font
	icon := ""
	secondaryIcon := ""

	useGopherFont := false

	plusrom := false
	supercharger := false
	coprocDev := false

	switch win.cartridgeNotice.event {
	case notifications.NotifySuperchargerSoundloadStarted:
		supercharger = true
		useGopherFont = true
		icon = fmt.Sprintf("%c", fonts.Tape)
		secondaryIcon = fmt.Sprintf("%c", fonts.TapePlay)
	case notifications.NotifySuperchargerSoundloadEnded:
		supercharger = true
		useGopherFont = true
		icon = fmt.Sprintf("%c", fonts.Tape)
		secondaryIcon = fmt.Sprintf("%c", fonts.TapeStop)
	case notifications.NotifySuperchargerSoundloadRewind:
		supercharger = true
		useGopherFont = true
		icon = fmt.Sprintf("%c", fonts.Tape)
		secondaryIcon = fmt.Sprintf("%c", fonts.TapeRewind)
	case notifications.NotifyPlusROMNetwork:
		plusrom = true
		useGopherFont = true
		secondaryIcon = ""
		icon = fmt.Sprintf("%c", fonts.Wifi)
	default:
		if ntfy.coprocDev {
			coprocDev = true
			useGopherFont = false
			icon = fmt.Sprintf("%c", fonts.Developer)
		} else {
			return
		}
	}

	// check preferences and return if the notification is not to be displayed
	if plusrom && !win.img.prefs.plusromNotifications.Get().(bool) {
		return
	}
	if supercharger && !win.img.prefs.superchargerNotifications.Get().(bool) {
		return
	}
	if coprocDev && !win.img.prefs.coprocDevNotification.Get().(bool) {
		return
	}

	dimen := win.img.plt.displaySize()
	pos := imgui.Vec2{dimen[0], 0}

	width := win.img.fonts.gopher2600IconsSize * 1.5
	if secondaryIcon != "" {
		width += win.img.fonts.largeFontAwesomeSize * 1.5
	}

	// position is based on which font we're using
	if useGopherFont {
		imgui.PushFont(win.img.fonts.gopher2600Icons)
		pos.X -= win.img.fonts.gopher2600IconsSize * 1.2
		if secondaryIcon != "" {
			pos.X -= win.img.fonts.largeFontAwesomeSize * 2.0
		}
	} else {
		imgui.PushFont(win.img.fonts.veryLargeFontAwesome)
		pos.X -= win.img.fonts.veryLargeFontAwesomeSize
		pos.X -= 20
		pos.Y += 10
	}
	defer imgui.PopFont()

	imgui.SetNextWindowPos(pos)
	imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.Transparent)
	defer imgui.PopStyleColorV(2)

	a := float32(win.img.prefs.notificationVisibility.Get().(float64))
	imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{a, a, a, a})
	defer imgui.PopStyleColor()

	imgui.BeginV("##cartridgeNotification", &ntfy.open, imgui.WindowFlagsAlwaysAutoResize|
		imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|imgui.WindowFlagsNoDecoration)

	imgui.Text(icon)

	imgui.SameLine()

	if secondaryIcon != "" {
		// position sub-icon so that it is centered vertically with the main icon
		dim := imgui.CursorScreenPos()
		dim.Y += (win.img.fonts.gopher2600IconsSize - win.img.fonts.largeFontAwesomeSize) * 0.5
		imgui.SetCursorScreenPos(dim)

		imgui.PushFont(win.img.fonts.largeFontAwesome)
		imgui.Text(secondaryIcon)
		imgui.PopFont()
	}

	imgui.End()
}
