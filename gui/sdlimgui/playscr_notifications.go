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
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
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

func (pn *peripheralNotification) set(peripheral plugging.PeripheralID) {
	pn.frames = notificationDurationPeripheral

	switch peripheral {
	case plugging.PeriphStick:
		pn.icon = fmt.Sprintf("%c", fonts.Stick)
	case plugging.PeriphPaddle:
		pn.icon = fmt.Sprintf("%c", fonts.Paddle)
	case plugging.PeriphKeypad:
		pn.icon = fmt.Sprintf("%c", fonts.Keypad)
	case plugging.PeriphSavekey:
		pn.icon = fmt.Sprintf("%c", fonts.Savekey)
	case plugging.PeriphGamepad:
		pn.icon = fmt.Sprintf("%c", fonts.Gamepad)
	case plugging.PeriphAtariVox:
		pn.icon = fmt.Sprintf("%c", fonts.AtariVox)
	default:
		pn.icon = ""
		return
	}
}

func (pn *peripheralNotification) tick() {
	pn.frames--
}

// pos should be the coordinate of the *extreme* bottom left or bottom right of
// the playscr window. the values will be adjusted according to whether we're
// display an icon or text.
func (pn *peripheralNotification) draw(win *playScr) {
	if pn.frames <= 0 {
		return
	}

	pn.tick()

	if !win.img.prefs.controllerNotifcations.Get().(bool) {
		return
	}

	// we'll be using the icon font for display in this window
	imgui.PushFont(win.img.glsl.fonts.gopher2600Icons)
	defer imgui.PopFont()

	// position window so that it is fully visible at the bottom of the screen.
	// taking special care of the right aligned window
	var id string
	var pos imgui.Vec2
	dimen := win.img.plt.displaySize()
	if pn.rightAlign {
		pos = imgui.Vec2{dimen[0], dimen[1]}
		id = "##controlleralertright"
		pos.X -= win.img.glsl.fonts.gopher2600IconsSize * 1.35
	} else {
		pos = imgui.Vec2{0, dimen[1]}
		id = "##controlleralertleft"
		pos.X += win.img.glsl.fonts.gopher2600IconsSize * 0.20
	}
	pos.Y -= win.img.glsl.fonts.gopher2600IconsSize * 1.35

	imgui.SetNextWindowPos(pos)
	imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.Transparent)

	periphOpen := true
	imgui.BeginV(id, &periphOpen, imgui.WindowFlagsAlwaysAutoResize|
		imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|
		imgui.WindowFlagsNoDecoration|imgui.WindowFlagsNoSavedSettings)

	imgui.Text(pn.icon)

	imgui.PopStyleColorV(2)
	imgui.End()
}

// emulationEventNotification is used to draw an indicator on the screen for
// events defined in the emulation package.
type emulationEventNotification struct {
	emulation    emulation.Emulation
	open         bool
	currentEvent emulation.Event
	frames       int

	// audio mute is handled differently to other events. we want the icon for
	// mute to always be shown unless another icon event has been received
	// since. when the previous event expires we want to reassign EventMute to
	// currentEvent
	mute bool
}

func (ee *emulationEventNotification) set(event emulation.Event) {
	ee.currentEvent = event
	ee.open = true
	ee.frames = notificationDurationEvent
	switch event {
	case emulation.EventRun:
		ee.frames = notificationDurationEventRun
	case emulation.EventScreenshot:
		ee.frames = notificationDurationScreenshot
	case emulation.EventMute:
		ee.mute = true
	case emulation.EventUnmute:
		ee.mute = false
	}
}

func (ee *emulationEventNotification) tick() {
	if !ee.open || ee.frames <= 0 {
		return
	}

	ee.frames--

	if ee.frames == 0 {
		// if emulation is paused then force the current event to EventPause
		if ee.emulation.State() == emulation.Paused {
			ee.currentEvent = emulation.EventPause
		}

		// special handling of open when current event is EventPause or if mute
		// is enabled
		if ee.currentEvent != emulation.EventPause {
			if ee.mute {
				ee.open = true
				ee.currentEvent = emulation.EventMute
			} else {
				ee.open = false
			}
		}
	}
}

func (ee *emulationEventNotification) draw(win *playScr, hosted bool) {
	if !ee.open {
		return
	}

	ee.tick()

	if !hosted {
		imgui.SetNextWindowPos(imgui.Vec2{X: 10, Y: 10})
		imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.Transparent)
		imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.Transparent)
		defer imgui.PopStyleColorV(2)

		imgui.BeginV("##cartridgeevent", &ee.open, imgui.WindowFlagsAlwaysAutoResize|
			imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|
			imgui.WindowFlagsNoDecoration|imgui.WindowFlagsNoSavedSettings|
			imgui.WindowFlagsNoBringToFrontOnFocus)
		defer imgui.End()

		imgui.PushFont(win.img.glsl.fonts.veryLargeFontAwesome)
		defer imgui.PopFont()
	}

	switch ee.currentEvent {
	case emulation.EventInitialising:
		imgui.Text("")
	case emulation.EventPause:
		imgui.Text(string(fonts.EmulationPause))
	case emulation.EventRun:
		imgui.Text(string(fonts.EmulationRun))
	case emulation.EventRewindBack:
		imgui.Text(string(fonts.EmulationRewindBack))
	case emulation.EventRewindFoward:
		imgui.Text(string(fonts.EmulationRewindForward))
	case emulation.EventRewindAtStart:
		imgui.Text(string(fonts.EmulationRewindAtStart))
	case emulation.EventRewindAtEnd:
		imgui.Text(string(fonts.EmulationRewindAtEnd))
	case emulation.EventScreenshot:
		imgui.Text(string(fonts.Camera))
	case emulation.EventMute:
		if hosted || win.img.prefs.audioMuteNotification.Get().(bool) {
			imgui.Text(string(fonts.AudioMute))
		}
	}
}

// cartridgeEventNotification is used to draw an indicator on the screen for cartridge
// events defined in the mapper package.
type cartridgeEventNotification struct {
	open         bool
	currentEvent mapper.Event
	frames       int
}

func (ce *cartridgeEventNotification) set(event mapper.Event) {
	ce.currentEvent = event
	switch ce.currentEvent {
	case mapper.EventSuperchargerSoundloadStarted:
		ce.open = true
	case mapper.EventSuperchargerSoundloadEnded:
		ce.frames = notificationDurationCartridge
	case mapper.EventSuperchargerSoundloadRewind:
		ce.frames = notificationDurationCartridge
	case mapper.EventPlusROMNetwork:
		ce.open = true
		ce.frames = notificationDurationCartridge
	}
}

func (ce *cartridgeEventNotification) tick() {
	if !ce.open || ce.frames <= 0 {
		return
	}

	ce.frames--

	if ce.frames == 0 {
		switch ce.currentEvent {
		case mapper.EventSuperchargerSoundloadEnded:
			ce.open = false
		case mapper.EventSuperchargerSoundloadRewind:
			ce.currentEvent = mapper.EventSuperchargerSoundloadStarted
		case mapper.EventPlusROMNetwork:
			ce.open = false
		}
	}
}

func (ce *cartridgeEventNotification) draw(win *playScr) {
	if !ce.open {
		return
	}

	ce.tick()

	// notifications are made up of an icon and a sub-icon. icons must be from
	// the gopher2600Icons font and the sub-icon from the largeFontAwesome font
	icon := ""
	subIcon := ""

	plusrom := false
	supercharger := false

	switch win.cartridgeEvent.currentEvent {
	case mapper.EventSuperchargerSoundloadStarted:
		supercharger = true
		icon = fmt.Sprintf("%c", fonts.Tape)
		subIcon = fmt.Sprintf("%c", fonts.TapePlay)
	case mapper.EventSuperchargerSoundloadEnded:
		supercharger = true
		icon = fmt.Sprintf("%c", fonts.Tape)
		subIcon = fmt.Sprintf("%c", fonts.TapeStop)
	case mapper.EventSuperchargerSoundloadRewind:
		supercharger = true
		icon = fmt.Sprintf("%c", fonts.Tape)
		subIcon = fmt.Sprintf("%c", fonts.TapeRewind)
	case mapper.EventPlusROMNetwork:
		plusrom = true
		icon = fmt.Sprintf("%c", fonts.Wifi)
	default:
		return
	}

	// check preferences and return if the notification is not to be displayed
	if plusrom && !win.img.prefs.plusromNotifications.Get().(bool) {
		return
	}
	if supercharger && !win.img.prefs.superchargerNotifications.Get().(bool) {
		return
	}

	dimen := win.img.plt.displaySize()
	pos := imgui.Vec2{dimen[0], 0}

	// position window so that it is right justified and shows entirity of window (calculated with
	// the knowledge that we're using two glyphs of fixed size)
	width := win.img.glsl.fonts.gopher2600IconsSize * 1.5
	if subIcon != "" {
		width += win.img.glsl.fonts.largeFontAwesomeSize * 1.5
	}
	pos.X -= width

	imgui.SetNextWindowPos(pos)
	imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.Transparent)

	imgui.BeginV("##cartridgeevent", &ce.open, imgui.WindowFlagsAlwaysAutoResize|
		imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|imgui.WindowFlagsNoDecoration)

	imgui.PushFont(win.img.glsl.fonts.gopher2600Icons)
	imgui.Text(icon)
	imgui.PopFont()

	imgui.SameLine()

	if subIcon != "" {
		// position sub-icon so that it is centered vertically with the main icon
		dim := imgui.CursorScreenPos()
		dim.Y += (win.img.glsl.fonts.gopher2600IconsSize - win.img.glsl.fonts.largeFontAwesomeSize) * 0.5
		imgui.SetCursorScreenPos(dim)

		imgui.PushFont(win.img.glsl.fonts.largeFontAwesome)
		imgui.Text(subIcon)
		imgui.PopFont()
	}

	imgui.PopStyleColorV(2)
	imgui.End()
}
