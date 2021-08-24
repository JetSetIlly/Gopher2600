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
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

const notificationDuration = 60 // frames

// peripheralNotification is used to draw an indicator on the screen for controller change events
type peripheralNotification struct {
	frames     int
	icon       string
	rightAlign bool
}

func (ca *peripheralNotification) set(peripheral plugging.PeripheralID) {
	ca.frames = notificationDuration

	switch peripheral {
	case plugging.PeriphStick:
		ca.icon = fmt.Sprintf("%c", fonts.Stick)
	case plugging.PeriphPaddle:
		ca.icon = fmt.Sprintf("%c", fonts.Paddle)
	case plugging.PeriphKeypad:
		ca.icon = fmt.Sprintf("%c", fonts.Keypad)
	case plugging.PeriphSavekey:
		ca.icon = fmt.Sprintf("%c", fonts.Savekey)
	default:
		ca.icon = ""
		return
	}

}

func (ca *peripheralNotification) tick() {
	ca.frames--
}

// pos should be the coordinate of the *extreme* bottom left or bottom right of
// the playscr window. the values will be adjusted according to whether we're
// display an icon or text.
func (ca *peripheralNotification) draw(win *playScr) {
	if ca.frames <= 0 {
		return
	}

	ca.tick()

	if !win.img.prefs.controllerNotifcations.Get().(bool) {
		return
	}

	// we'll be using the icon font for display in this window
	imgui.PushFont(win.img.glsl.gopher2600Icons)
	defer imgui.PopFont()

	// position window so that it is fully visible at the bottom of the screen.
	// taking special care of the right aligned window
	var id string
	var pos imgui.Vec2
	dimen := win.img.plt.displaySize()
	if ca.rightAlign {
		pos = imgui.Vec2{dimen[0], dimen[1]}
		id = "##controlleralertright"
		pos.X -= win.img.glsl.gopher2600IconsSize * 1.5
	} else {
		pos = imgui.Vec2{0, dimen[1]}
		id = "##controlleralertleft"
	}
	pos.Y -= win.img.glsl.gopher2600IconsSize * 1.5

	imgui.SetNextWindowPos(pos)
	imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.Transparent)

	imgui.BeginV(id, &win.fpsOpen, imgui.WindowFlagsAlwaysAutoResize|
		imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|imgui.WindowFlagsNoDecoration)

	imgui.Text(ca.icon)

	imgui.PopStyleColorV(2)
	imgui.End()
}

// cartridgeEventNotification is used to draw an indicator on the screen for cartride
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
		ce.frames = notificationDuration
	case mapper.EventSuperchargerSoundloadRewind:
		ce.frames = notificationDuration
	case mapper.EventPlusROMNetwork:
		ce.open = true
		ce.frames = notificationDuration
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
	width := win.img.glsl.gopher2600IconsSize * 1.5
	if subIcon != "" {
		width += win.img.glsl.largeFontAwesomeSize * 1.5
	}
	pos.X -= width

	imgui.SetNextWindowPos(pos)
	imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.Transparent)

	imgui.BeginV("##cartridgeevent", &ce.open, imgui.WindowFlagsAlwaysAutoResize|
		imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|imgui.WindowFlagsNoDecoration)

	imgui.PushFont(win.img.glsl.gopher2600Icons)
	imgui.Text(icon)
	imgui.PopFont()

	imgui.SameLine()

	if subIcon != "" {
		// position sub-icon so that it is centered vertically with the main icon
		dim := imgui.CursorScreenPos()
		dim.Y += (win.img.glsl.gopher2600IconsSize - win.img.glsl.largeFontAwesomeSize) * 0.5
		imgui.SetCursorScreenPos(dim)

		imgui.PushFont(win.img.glsl.largeFontAwesome)
		imgui.Text(subIcon)
		imgui.PopFont()
	}

	imgui.PopStyleColorV(2)
	imgui.End()
}
