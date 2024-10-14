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
	"runtime"
	"strings"
	"time"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/notifications"
)

type overlayLatch int

const (
	overlayLatchPinned = -1
	overlayLatchOff    = 0
	overlayLatchBrief  = 30
	overlayLatchShort  = 60
	overlayLatchLong   = 90
)

func (ct *overlayLatch) forceExpire() {
	*ct = overlayLatchOff
}

// reduces the duration value. returns false if count has expired. if the
// duration has been "pinned" then value will return true
func (ct *overlayLatch) tick() bool {
	if *ct == overlayLatchOff {
		return false
	}
	if *ct == overlayLatchPinned {
		return true
	}
	*ct = *ct - 1
	return true
}

// returns true if duration is not off or pinned
func (ct *overlayLatch) expired() bool {
	return *ct != overlayLatchPinned && *ct == overlayLatchOff
}

type playscrOverlay struct {
	playscr *playScr

	// fps information is updated on every pule of the time.Ticker. the fps and
	// refreshRate string is updated at that time and then displayed on every draw()
	fpsPulse    *time.Ticker
	fpsForce    chan bool
	fps         string
	refreshRate string

	// memory stats are updated along with the fpsPulse
	memStats runtime.MemStats

	// top-left corner of the overlay includes emulation state. if the
	// "fpsOverlay" is active then these will be drawn alongside the FPS
	// information
	state      govern.State
	subState   govern.SubState
	stateLatch overlayLatch

	// events are user-activated events and require immediate feedback
	event      notifications.Notice
	eventLatch overlayLatch

	// icons in the top-left corner of the overlay are drawn according to a
	// priority. the iconQueue list the icons to be drawn in order
	iconQueue []rune

	// top-right corner of the overlay
	cartridge      notifications.Notice
	cartridgeLatch overlayLatch

	// bottom-left corner of the overlay
	leftPort      plugging.PeripheralID
	leftPortLatch overlayLatch

	// bottom-right corner of the overlay
	rightPort      plugging.PeripheralID
	rightPortLatch overlayLatch

	// visibility of icons is set from the preferences once per draw()
	visibility float32
}

const overlayPadding = 10

func (oly *playscrOverlay) set(v any, args ...any) {
	switch n := v.(type) {
	case plugging.PortID:
		switch n {
		case plugging.PortLeft:
			oly.leftPort = args[0].(plugging.PeripheralID)
			oly.leftPortLatch = overlayLatchShort
		case plugging.PortRight:
			oly.rightPort = args[0].(plugging.PeripheralID)
			oly.rightPortLatch = overlayLatchShort
		}
	case notifications.Notice:
		switch n {
		case notifications.NotifySuperchargerSoundloadStarted:
			oly.cartridge = n
			oly.cartridgeLatch = overlayLatchPinned
		case notifications.NotifySuperchargerSoundloadEnded:
			oly.cartridge = n
			oly.cartridgeLatch = overlayLatchShort
		case notifications.NotifySuperchargerSoundloadRewind:
			return

		case notifications.NotifyPlusROMNetwork:
			oly.cartridge = n
			oly.cartridgeLatch = overlayLatchShort

		case notifications.NotifyScreenshot:
			oly.event = n
			oly.eventLatch = overlayLatchShort

		default:
			return
		}
	}
}

func (oly *playscrOverlay) draw() {
	imgui.PushStyleColor(imgui.StyleColorWindowBg, oly.playscr.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorBorder, oly.playscr.img.cols.Transparent)
	defer imgui.PopStyleColorV(2)

	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.Vec2{})
	defer imgui.PopStyleVarV(1)

	imgui.SetNextWindowPos(imgui.Vec2{X: 0, Y: 0})

	sz := oly.playscr.img.plt.displaySize()
	imgui.SetNextWindowSize(imgui.Vec2{X: sz[0], Y: sz[1]})

	imgui.BeginV("##playscrOverlay", nil, imgui.WindowFlagsAlwaysAutoResize|
		imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|
		imgui.WindowFlagsNoDecoration|imgui.WindowFlagsNoSavedSettings|
		imgui.WindowFlagsNoBringToFrontOnFocus)
	defer imgui.End()

	oly.visibility = float32(oly.playscr.img.prefs.notificationVisibility.Get().(float64))

	oly.drawTopLeft()
	oly.drawTopRight()
	oly.drawBottomLeft()
	oly.drawBottomRight()
}

func (oly *playscrOverlay) updateRefreshRate() {
	fps, refreshRate := oly.playscr.img.dbg.VCS().TV.GetActualFPS()
	oly.fps = fmt.Sprintf("%03.2f fps", fps)
	oly.refreshRate = fmt.Sprintf("%03.2fhz", refreshRate)
}

// information in the top left corner of the overlay are about the emulation.
// eg. whether audio is mute, or the emulation is paused, etc. it is also used
// to display the FPS counter and other TV information
func (oly *playscrOverlay) drawTopLeft() {
	pos := imgui.CursorScreenPos()
	pos.X += overlayPadding
	pos.Y += overlayPadding

	// by default only one icon is shown in the top left corner. however, if the
	// FPS overlay is being used we use the space to draw smaller icons
	var useIconQueue bool

	// draw FPS information if it's enabled
	if oly.playscr.img.prefs.fpsDetail.Get().(bool) {
		// it's easier if we put topleft of overlay in a window because the window
		// will control the width and positioning automatically. if we don't then
		// the horizntal rules will stretch the width of the screen and each new line of
		// text in the fps detail will need to be repositioned for horizontal
		// padding
		imgui.SetNextWindowPos(pos)
		imgui.BeginV("##fpsDetail", nil, imgui.WindowFlagsAlwaysAutoResize|
			imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|
			imgui.WindowFlagsNoDecoration|imgui.WindowFlagsNoSavedSettings|
			imgui.WindowFlagsNoBringToFrontOnFocus)
		defer imgui.End()

		select {
		case <-oly.fpsPulse.C:
			oly.updateRefreshRate()
			runtime.ReadMemStats(&oly.memStats)
		default:
		}

		imgui.Text(fmt.Sprintf("Emulation: %s", oly.fps))
		fr := imgui.CurrentIO().Framerate()
		if fr == 0.0 {
			imgui.Text("Rendering: waiting")
		} else {
			imgui.Text(fmt.Sprintf("Rendering: %03.2f fps", fr))
		}

		imguiSeparator()

		if coproc := oly.playscr.img.cache.VCS.Mem.Cart.GetCoProc(); coproc != nil {
			clk := float32(oly.playscr.img.dbg.VCS().Env.Prefs.ARM.Clock.Get().(float64))
			imgui.Text(fmt.Sprintf("%s Clock: %.0f Mhz", coproc.ProcessorID(), clk))
			imguiSeparator()
		}

		imgui.Text(fmt.Sprintf("%.1fx scaling", oly.playscr.scaling))
		imgui.Text(fmt.Sprintf("%d total scanlines", oly.playscr.scr.crit.frameInfo.TotalScanlines))

		imguiSeparator()

		// this construct (spacing followed by a same-line directive) is only
		// necessary so that the extreme left pixel of the VBLANKtop icon is not
		// chopped off. it's a very small detail but worth doing
		imgui.Spacing()
		imgui.SameLineV(0, 1)

		vblankBounds := fmt.Sprintf("%c %d  %c %d",
			fonts.VBLANKtop,
			oly.playscr.scr.crit.frameInfo.VBLANKtop,
			fonts.VBLANKbottom,
			oly.playscr.scr.crit.frameInfo.VBLANKbottom)
		vblankBounds = strings.ReplaceAll(vblankBounds, "-1", "-")
		imgui.Text(vblankBounds)
		if oly.playscr.scr.crit.frameInfo.VBLANKunstable {
			imgui.SameLineV(0, 5)
			imgui.Text(string(fonts.Bug))
		}
		if oly.playscr.scr.crit.frameInfo.VBLANKatari {
			imgui.SameLineV(0, 15)
			imgui.Text(string(fonts.VBLANKatari))
		}

		imgui.Spacing()
		if oly.playscr.scr.crit.frameInfo.FromVSYNC {
			imgui.Text(fmt.Sprintf("VSYNC %d+%d", oly.playscr.scr.crit.frameInfo.VSYNCscanline,
				oly.playscr.scr.crit.frameInfo.VSYNCcount))
			if oly.playscr.scr.crit.frameInfo.VSYNCunstable {
				imgui.SameLineV(0, 5)
				imgui.Text(string(fonts.Bug))
			}
		} else {
			imgui.Text(fmt.Sprintf("VSYNC %c", fonts.Bug))
		}

		imguiSeparator()
		imgui.Text(oly.playscr.img.screen.crit.frameInfo.Spec.ID)

		imgui.SameLine()
		imgui.Text(oly.refreshRate)

		imguiSeparator()
		imgui.Text(fmt.Sprintf("%d frame input lag", oly.playscr.scr.crit.frameQueueLen))
		if oly.playscr.scr.nudgeIconCt > 0 {
			imgui.SameLine()
			imgui.Text(string(fonts.Nudge))
		}

		if oly.playscr.img.prefs.memoryUsageInOverlay.Get().(bool) {
			imguiSeparator()
			imgui.Text(fmt.Sprintf("Alloc = %v MB\n", oly.memStats.Alloc/1048576))
			imgui.Text(fmt.Sprintf(" TotalAlloc = %v MB\n", oly.memStats.TotalAlloc/1048576))
			imgui.Text(fmt.Sprintf(" Sys = %v MB\n", oly.memStats.Sys/1048576))
			imgui.Text(fmt.Sprintf(" NumGC = %v", oly.memStats.NumGC))
		}

		// create space in the window for any icons that we might want to draw.
		// what's good about this is that it makes sure that the window is large
		// enough from frame-to-frame. without this, there will be a visble
		// delay when the window is resized
		imgui.Spacing()
		p := imgui.CursorScreenPos()
		imgui.Text("")
		imgui.SetCursorScreenPos(p)

		// draw developer icon if BorrowSource() returns a non-nil value
		oly.playscr.img.dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
			if src != nil {
				imgui.Text(string(fonts.Developer))
				imgui.SameLine()
			}
		})

		// we can draw multiple icons if required
		useIconQueue = true

	} else {
		// we'll only be drawing one icon so we only need to set the cursor
		// position once, so there's no need for a window as would be the case
		// if fps detail was activated
		imgui.SetCursorScreenPos(pos)

		// FPS overlay is not active so we increase the font size for any icons
		// that may be drawn hereafter in this window
		imgui.PushFont(oly.playscr.img.fonts.veryLargeFontAwesome)
		defer imgui.PopFont()

		// add visibility adjustment if there is no FPS overlay
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: oly.visibility, Y: oly.visibility, Z: oly.visibility, W: oly.visibility})
		defer imgui.PopStyleColor()
	}

	// start a new icons queue
	oly.iconQueue = oly.iconQueue[:0]

	// mute is likely to be the icon visible the longest so has the lowest priority
	if oly.playscr.img.prefs.audioMutePlaymode.Get().(bool) && oly.playscr.img.prefs.audioMuteNotification.Get().(bool) {
		oly.iconQueue = append(oly.iconQueue, fonts.AudioMute)
	}

	// the real current state as set by the emulation is used to decide what
	// state to use for the overlay icon
	state := oly.playscr.img.dbg.State()
	subState := oly.playscr.img.dbg.SubState()

	switch state {
	case govern.Paused:
		// handling the pause state is the trickiest to get right. we want to
		// prioritise the pause icon in some cases but not in others
		switch oly.state {
		case govern.Rewinding:
			// if the previous state was the rewinding state a pause icon will
			// show if the pause sub-state is not normal or if the
			// previous state latch has expired
			if subState != govern.Normal || oly.stateLatch.expired() {
				oly.state = state
				oly.subState = subState
				oly.stateLatch = overlayLatchPinned
			}
		default:
			oly.state = state
			oly.subState = subState
			oly.stateLatch = overlayLatchPinned
		}
	case govern.Running:
		if state != oly.state {
			oly.state = state
			oly.subState = subState
			oly.stateLatch = overlayLatchShort
		}
	case govern.Rewinding:
		oly.state = state
		oly.subState = subState

		// refresh how the hold duration on every render frame that the
		// rewinding state is seen. this is so that the duration of the rewind
		// icon doesn't expire causing the pause icon to appear every so often
		//
		// (the way rewinding is implemented in the emulation means that the
		// rewinding state is interspersed very quickly with the paused state.
		// that works great for internal emulation purposes but requires careful
		// handling for UI purposes)
		oly.stateLatch = overlayLatchBrief
	}

	// the state duration is ticked and the icon is shown unless the tick has
	// expired (returns false)
	if oly.stateLatch.tick() {
		switch oly.state {
		case govern.Paused:
			switch oly.subState {
			case govern.PausedAtStart:
				oly.iconQueue = append(oly.iconQueue, fonts.EmulationPausedAtStart)
			case govern.PausedAtEnd:
				oly.iconQueue = append(oly.iconQueue, fonts.EmulationPausedAtEnd)
			default:
				oly.iconQueue = append(oly.iconQueue, fonts.EmulationPause)
			}
		case govern.Running:
			oly.iconQueue = append(oly.iconQueue, fonts.EmulationRun)
		case govern.Rewinding:
			switch oly.subState {
			case govern.RewindingBackwards:
				oly.iconQueue = append(oly.iconQueue, fonts.EmulationRewindBack)
			case govern.RewindingForwards:
				oly.iconQueue = append(oly.iconQueue, fonts.EmulationRewindForward)
			default:
			}
		}
	}

	// events have the highest priority. we can think of these as user activated
	// events, such as the triggering of a screenshot. we therefore want to give
	// the user confirmation feedback immediately over other icons
	if oly.eventLatch.tick() {
		switch oly.event {
		case notifications.NotifyScreenshot:
			oly.iconQueue = append(oly.iconQueue, fonts.Camera)
		}
	}

	// draw only the last (ie. most important) icon unless the icon queue flag
	// has been set
	if !useIconQueue {
		if len(oly.iconQueue) > 0 {
			imgui.Text(string(oly.iconQueue[len(oly.iconQueue)-1]))
		}
		return
	}

	// draw icons in order of priority
	for _, i := range oly.iconQueue {
		imgui.Text(string(i))
		imgui.SameLine()
	}
	return
}

// information in the top right of the overlay is about the cartridge. ie.
// information from the cartridge about what is happening. for example,
// supercharger tape activity, or PlusROM network activity, etc.
func (oly *playscrOverlay) drawTopRight() {
	if !oly.cartridgeLatch.tick() {
		return
	}

	var icon string
	var secondaryIcon string

	switch oly.cartridge {
	case notifications.NotifySuperchargerSoundloadStarted:
		if oly.playscr.img.prefs.superchargerNotifications.Get().(bool) {
			icon = fmt.Sprintf("%c", fonts.Tape)
			secondaryIcon = fmt.Sprintf("%c", fonts.TapePlay)
		}
	case notifications.NotifySuperchargerSoundloadEnded:
		if oly.playscr.img.prefs.superchargerNotifications.Get().(bool) {
			icon = fmt.Sprintf("%c", fonts.Tape)
			secondaryIcon = fmt.Sprintf("%c", fonts.TapeStop)
		}
	case notifications.NotifySuperchargerSoundloadRewind:
		if oly.playscr.img.prefs.superchargerNotifications.Get().(bool) {
			icon = fmt.Sprintf("%c", fonts.Tape)
			secondaryIcon = fmt.Sprintf("%c", fonts.TapeRewind)
		}
	case notifications.NotifyPlusROMNetwork:
		if oly.playscr.img.prefs.plusromNotifications.Get().(bool) {
			icon = fmt.Sprintf("%c", fonts.Wifi)
		}
	default:
		return
	}

	pos := imgui.WindowContentRegionMax()
	pos.X -= oly.playscr.img.fonts.gopher2600IconsSize + overlayPadding
	pos.Y = 0
	if secondaryIcon != "" {
		pos.X -= oly.playscr.img.fonts.largeFontAwesomeSize * 2
	}

	imgui.PushFont(oly.playscr.img.fonts.gopher2600Icons)
	defer imgui.PopFont()

	imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: oly.visibility, Y: oly.visibility, Z: oly.visibility, W: oly.visibility})
	defer imgui.PopStyleColor()

	imgui.SetCursorScreenPos(pos)
	imgui.Text(icon)

	if secondaryIcon != "" {
		imgui.PushFont(oly.playscr.img.fonts.largeFontAwesome)
		defer imgui.PopFont()

		imgui.SameLine()
		pos = imgui.CursorScreenPos()
		pos.Y += (oly.playscr.img.fonts.gopher2600IconsSize - oly.playscr.img.fonts.largeFontAwesomeSize) * 0.5

		imgui.SetCursorScreenPos(pos)
		imgui.Text(secondaryIcon)
	}
}

func (oly *playscrOverlay) drawBottomLeft() {
	if !oly.leftPortLatch.tick() {
		return
	}

	if !oly.playscr.img.prefs.controllerNotifcations.Get().(bool) {
		return
	}

	pos := imgui.WindowContentRegionMax()
	pos.X = overlayPadding
	pos.Y -= oly.playscr.img.fonts.gopher2600IconsSize + overlayPadding

	imgui.SetCursorScreenPos(pos)
	oly.drawPeripheral(oly.leftPort)
}

func (oly *playscrOverlay) drawBottomRight() {
	if !oly.rightPortLatch.tick() {
		return
	}

	if !oly.playscr.img.prefs.controllerNotifcations.Get().(bool) {
		return
	}

	pos := imgui.WindowContentRegionMax()
	pos.X -= oly.playscr.img.fonts.gopher2600IconsSize + overlayPadding
	pos.Y -= oly.playscr.img.fonts.gopher2600IconsSize + overlayPadding

	imgui.SetCursorScreenPos(pos)
	oly.drawPeripheral(oly.rightPort)
}

// drawPeripheral is used to draw the peripheral in the bottom left and bottom
// right corners of the overlay
func (oly *playscrOverlay) drawPeripheral(peripID plugging.PeripheralID) {
	imgui.PushFont(oly.playscr.img.fonts.gopher2600Icons)
	defer imgui.PopFont()

	imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: oly.visibility, Y: oly.visibility, Z: oly.visibility, W: oly.visibility})
	defer imgui.PopStyleColor()

	switch peripID {
	case plugging.PeriphStick:
		imgui.Text(fmt.Sprintf("%c", fonts.Stick))
	case plugging.PeriphPaddles:
		imgui.Text(fmt.Sprintf("%c", fonts.Paddle))
	case plugging.PeriphKeypad:
		imgui.Text(fmt.Sprintf("%c", fonts.Keypad))
	case plugging.PeriphSavekey:
		imgui.Text(fmt.Sprintf("%c", fonts.Savekey))
	case plugging.PeriphGamepad:
		imgui.Text(fmt.Sprintf("%c", fonts.Gamepad))
	case plugging.PeriphAtariVox:
		imgui.Text(fmt.Sprintf("%c", fonts.AtariVox))
	}
}
