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
	"strings"

	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/gui/sdlaudio"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/notifications"
	"github.com/jetsetilly/imgui-go/v5"
)

type overlayLatch struct {
	duration int
	delay    int
}

const (
	overlayLatchPinned = -1
	overlayLatchOff    = 0
	overlayLatchBrief  = 30
	overlayLatchShort  = 60
)

// reduces the duration value. returns false if count has expired. if the
// duration has been "pinned" then value will return true
func (ct *overlayLatch) tick() bool {
	if ct.delay > 0 {
		ct.delay--
		return false
	}
	if ct.duration == overlayLatchOff {
		return false
	}
	if ct.duration == overlayLatchPinned {
		return true
	}
	ct.duration = ct.duration - 1
	return true
}

// returns true if duration is not off or pinned
func (ct *overlayLatch) expired() bool {
	return ct.duration != overlayLatchPinned && ct.duration == overlayLatchOff
}

type playscrOverlay struct {
	img     *SdlImgui
	playscr *playScr

	fps         string
	refreshRate string

	renderAlert int

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

func (o *playscrOverlay) set(v any, args ...any) {
	switch n := v.(type) {
	case plugging.PortID:
		switch n {
		case plugging.PortLeft:
			o.leftPort = args[0].(plugging.PeripheralID)
			o.leftPortLatch = overlayLatch{duration: overlayLatchShort, delay: 20}
		case plugging.PortRight:
			o.rightPort = args[0].(plugging.PeripheralID)
			o.rightPortLatch = overlayLatch{duration: overlayLatchShort, delay: 20}
		}
	case notifications.Notice:
		switch n {
		case notifications.NotifySuperchargerSoundloadStarted:
			o.cartridge = n
			o.cartridgeLatch = overlayLatch{duration: overlayLatchPinned}
		case notifications.NotifySuperchargerSoundloadEnded:
			o.cartridge = n
			o.cartridgeLatch = overlayLatch{duration: overlayLatchShort}
		case notifications.NotifySuperchargerSoundloadRewind:
			return

		case notifications.NotifyPlusROMNetwork:
			o.cartridge = n
			o.cartridgeLatch = overlayLatch{duration: overlayLatchShort}

		case notifications.NotifyScreenshot:
			o.event = n
			o.eventLatch = overlayLatch{duration: overlayLatchShort}

		default:
			return
		}
	}
}

func (o *playscrOverlay) draw(posMin imgui.Vec2, posMax imgui.Vec2) {
	imgui.PushStyleColor(imgui.StyleColorWindowBg, o.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorBorder, o.img.cols.Transparent)
	defer imgui.PopStyleColorV(2)

	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.Vec2{})
	defer imgui.PopStyleVarV(1)

	sz := posMax.Minus(posMin)
	imgui.SetNextWindowPos(posMin)
	imgui.SetNextWindowSize(sz)

	imgui.BeginV("##playscrOverlay", nil, imgui.WindowFlagsAlwaysAutoResize|
		imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|
		imgui.WindowFlagsNoDecoration|imgui.WindowFlagsNoSavedSettings|
		imgui.WindowFlagsNoBringToFrontOnFocus)
	defer imgui.End()

	o.visibility = float32(o.img.prefs.notificationVisibility.Get().(float64))

	o.drawTopLeft(posMin, posMax)
	o.drawTopRight(posMin, posMax)
	o.drawBottomLeft(posMin, posMax)
	o.drawBottomRight(posMin, posMax)
}

func (o *playscrOverlay) updateRefreshRate() {
	fps, refreshRate := o.img.dbg.VCS().TV.GetActualFPS()
	if fps == 0 {
		o.fps = "waiting"
	} else {
		o.fps = fmt.Sprintf("%03.2f fps", fps)
	}
	if refreshRate == 0 {
		o.refreshRate = "waiting"
	} else {
		o.refreshRate = fmt.Sprintf("%03.2fhz", refreshRate)
	}
}

// information in the top left corner of the overlay are about the emulation.
// eg. whether audio is mute, or the emulation is paused, etc. it is also used
// to display the FPS counter and other TV information
func (o *playscrOverlay) drawTopLeft(posMin imgui.Vec2, posMax imgui.Vec2) {
	pos := posMin
	imgui.SetCursorScreenPos(pos)
	pos.X += overlayPadding
	pos.Y += overlayPadding

	// by default only one icon is shown in the top left corner. however, if the
	// FPS overlay is being used we use the space to draw smaller icons
	var useIconQueue bool

	// draw FPS information if it's enabled
	if o.img.prefs.fpsDetail.Get().(bool) {
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

		o.updateRefreshRate()

		imgui.Textf("Emulation: %s", o.fps)
		r := imgui.CurrentIO().Framerate()
		if r == 0.0 {
			imgui.Text("Rendering: waiting")
		} else {
			imgui.Textf("Rendering: %03.2f fps", r)
		}

		imguiSeparator()

		if coproc := o.img.cache.VCS.Mem.Cart.GetCoProc(); coproc != nil {
			clk := float32(o.img.dbg.VCS().Env.Prefs.ARM.Clock.Get().(float64))
			imgui.Text(fmt.Sprintf("%s Clock: %.0f Mhz", coproc.ProcessorID(), clk))
			imguiSeparator()
		}

		imgui.Text(fmt.Sprintf("%.1fx scaling", o.playscr.scaling))
		imgui.Text(fmt.Sprintf("%d total scanlines", o.playscr.scr.crit.frameInfo.TotalScanlines))

		imguiSeparator()

		// this construct (spacing followed by a same-line directive) is only
		// necessary so that the extreme left pixel of the VBLANKtop icon is not
		// chopped off. it's a very small detail but worth doing
		imgui.Spacing()
		imgui.SameLineV(0, 1)

		vblankBounds := fmt.Sprintf("%c %d  %c %d",
			fonts.VBLANKtop,
			o.playscr.scr.crit.frameInfo.VBLANKtop,
			fonts.VBLANKbottom,
			o.playscr.scr.crit.frameInfo.VBLANKbottom)
		vblankBounds = strings.ReplaceAll(vblankBounds, "-1", "-")
		imgui.Text(vblankBounds)
		if o.playscr.scr.crit.frameInfo.VBLANKunstable {
			imgui.SameLineV(0, 5)
			imgui.Text(string(fonts.Bug))
		}
		if o.playscr.scr.crit.frameInfo.AtariSafe() {
			imgui.SameLineV(0, 15)
			imgui.Text(string(fonts.VBLANKatari))
		}

		imgui.Spacing()
		if o.playscr.scr.crit.frameInfo.FromVSYNC {
			imgui.Text(fmt.Sprintf("VSYNC %d+%d", o.playscr.scr.crit.frameInfo.VSYNCscanline,
				o.playscr.scr.crit.frameInfo.VSYNCcount))
			if o.playscr.scr.crit.frameInfo.VSYNCunstable {
				imgui.SameLineV(0, 5)
				imgui.Text(string(fonts.Bug))
			}
		} else {
			imgui.Text(fmt.Sprintf("VSYNC %c", fonts.Bug))
		}

		imguiSeparator()
		imgui.Text(o.img.screen.crit.frameInfo.Spec.ID)

		imgui.SameLine()
		imgui.Text(o.refreshRate)

		if o.img.prefs.frameQueueMeterInOverlay.Get().(bool) {
			imguiSeparator()

			imgui.PushStyleColor(imgui.StyleColorText, o.img.cols.FrameQueueSlackActive)
			for range o.playscr.scr.frameQueueSlack {
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
			}

			imgui.PushStyleColor(imgui.StyleColorText, o.img.cols.FrameQueueSlackInactive)
			for range o.playscr.scr.crit.frameQueueLen - o.playscr.scr.frameQueueSlack {
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
			}
			imgui.Text("")

			imgui.PopStyleColorV(2)

			imgui.Spacing()
			imgui.Textf("%2.2fms/frame", float32(o.img.plt.renderAvgTime.Nanoseconds())/1000000)
			if o.img.plt.renderAlert {
				o.renderAlert = 60
			} else if o.renderAlert > 0 {
				o.renderAlert--
			}
			if o.renderAlert > 0 {
				imgui.SameLineV(0, 5)
				imgui.Text(string(fonts.RenderTime))
			}
		}

		if o.img.prefs.audioQueueMeterInOverlay.Get().(bool) {
			// draw separator if there is no frame queue meter
			if !o.img.prefs.frameQueueMeterInOverlay.Get().(bool) {
				imguiSeparator()
			} else {
				imgui.Spacing()
			}

			queuedBytes := o.img.audio.QueuedBytes.Load()

			if queuedBytes == 0 {
				imgui.PushStyleColor(imgui.StyleColorText, o.img.cols.AudioQueueInactive)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.PopStyleColor()
			} else if queuedBytes < sdlaudio.QueueOkay {
				imgui.PushStyleColor(imgui.StyleColorText, o.img.cols.AudioQueueActive)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.PushStyleColor(imgui.StyleColorText, o.img.cols.AudioQueueInactive)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.PopStyleColorV(2)
			} else if queuedBytes < sdlaudio.QueueWarning {
				imgui.PushStyleColor(imgui.StyleColorText, o.img.cols.AudioQueueActive)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.PushStyleColor(imgui.StyleColorText, o.img.cols.AudioQueueInactive)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.PopStyleColorV(2)
			} else {
				imgui.PushStyleColor(imgui.StyleColorText, o.img.cols.AudioQueueActive)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.PopStyleColor()
			}

			imgui.Spacing()
			if !o.img.prefs.audioMutePlaymode.Get().(bool) {
				imgui.Textf("%dkb audio queue", queuedBytes/1024)
			}
		}

		if o.img.prefs.memoryUsageInOverlay.Get().(bool) {
			o.img.metrics.draw()
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
		o.img.dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
			if src != nil {
				imgui.Text(string(fonts.Developer))
				imgui.SameLine()
			}
		})

		// we can draw multiple icons if required
		useIconQueue = true
	}

	// start a new icons queue
	o.iconQueue = o.iconQueue[:0]

	// mute is likely to be the icon visible the longest so has the lowest priority
	if o.img.prefs.audioMutePlaymode.Get().(bool) && o.img.prefs.audioMuteNotification.Get().(bool) {
		o.iconQueue = append(o.iconQueue, fonts.AudioMute)
	}

	// the real current state as set by the emulation is used to decide what
	// state to use for the overlay icon
	state := o.img.dbg.State()
	subState := o.img.dbg.SubState()

	switch state {
	case govern.Paused:
		// handling the pause state is the trickiest to get right. we want to
		// prioritise the pause icon in some cases but not in others
		switch o.state {
		case govern.Rewinding:
			// if the previous state was the rewinding state a pause icon will
			// show if the pause sub-state is not normal or if the
			// previous state latch has expired
			if subState != govern.Normal || o.stateLatch.expired() {
				o.state = state
				o.subState = subState
				o.stateLatch = overlayLatch{duration: overlayLatchPinned}
			}
		default:
			o.state = state
			o.subState = subState
			o.stateLatch = overlayLatch{duration: overlayLatchPinned}
		}
	case govern.Running:
		if state != o.state {
			o.state = state
			o.subState = subState
			o.stateLatch = overlayLatch{duration: overlayLatchShort}
		}
	case govern.Rewinding:
		o.state = state
		o.subState = subState

		// refresh how the hold duration on every render frame that the
		// rewinding state is seen. this is so that the duration of the rewind
		// icon doesn't expire causing the pause icon to appear every so often
		//
		// (the way rewinding is implemented in the emulation means that the
		// rewinding state is interspersed very quickly with the paused state.
		// that works great for internal emulation purposes but requires careful
		// handling for UI purposes)
		o.stateLatch = overlayLatch{duration: overlayLatchBrief}
	}

	// the state duration is ticked and the icon is shown unless the tick has
	// expired (returns false)
	if o.stateLatch.tick() {
		switch o.state {
		case govern.Paused:
			switch o.subState {
			case govern.PausedAtStart:
				o.iconQueue = append(o.iconQueue, fonts.EmulationPausedAtStart)
			case govern.PausedAtEnd:
				o.iconQueue = append(o.iconQueue, fonts.EmulationPausedAtEnd)
			default:
				o.iconQueue = append(o.iconQueue, fonts.EmulationPause)
			}
		case govern.Running:
			o.iconQueue = append(o.iconQueue, fonts.EmulationRun)
		case govern.Rewinding:
			switch o.subState {
			case govern.RewindingBackwards:
				o.iconQueue = append(o.iconQueue, fonts.EmulationRewindBack)
			case govern.RewindingForwards:
				o.iconQueue = append(o.iconQueue, fonts.EmulationRewindForward)
			default:
			}
		}
	}

	// events have the highest priority. we can think of these as user activated
	// events, such as the triggering of a screenshot. we therefore want to give
	// the user confirmation feedback immediately over other icons
	if o.eventLatch.tick() {
		switch o.event {
		case notifications.NotifyScreenshot:
			o.iconQueue = append(o.iconQueue, fonts.Camera)
		}
	}

	// draw only the last (ie. most important) icon unless the icon queue flag
	// has been set
	if !useIconQueue {
		if len(o.iconQueue) > 0 {
			// we'll only be drawing one icon so we only need to set the cursor
			// position once, so there's no need for a window as would be the case
			// if fps detail was activated
			imgui.SetCursorScreenPos(pos)

			// FPS overlay is not active so we increase the font size for any icons
			// that may be drawn hereafter in this window
			imgui.PushFont(o.img.fonts.veryLargeFontAwesome)
			defer imgui.PopFont()

			// add visibility adjustment if there is no FPS overlay
			imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: o.visibility, Y: o.visibility, Z: o.visibility, W: o.visibility})
			defer imgui.PopStyleColor()
		}

		if len(o.iconQueue) > 0 {
			imgui.Text(string(o.iconQueue[len(o.iconQueue)-1]))
		}
		return
	}

	// draw icons in order of priority
	for _, i := range o.iconQueue {
		imgui.Text(string(i))
		imgui.SameLine()
	}
}

// information in the top right of the overlay is about the cartridge. ie.
// information from the cartridge about what is happening. for example,
// supercharger tape activity, or PlusROM network activity, etc.
func (o *playscrOverlay) drawTopRight(posMin imgui.Vec2, posMax imgui.Vec2) {
	if !o.cartridgeLatch.tick() {
		return
	}

	var icon string
	var secondaryIcon string

	switch o.cartridge {
	case notifications.NotifySuperchargerSoundloadStarted:
		if o.img.prefs.superchargerNotifications.Get().(bool) {
			icon = fmt.Sprintf("%c", fonts.Tape)
			secondaryIcon = fmt.Sprintf("%c", fonts.TapePlay)
		}
	case notifications.NotifySuperchargerSoundloadEnded:
		if o.img.prefs.superchargerNotifications.Get().(bool) {
			icon = fmt.Sprintf("%c", fonts.Tape)
			secondaryIcon = fmt.Sprintf("%c", fonts.TapeStop)
		}
	case notifications.NotifySuperchargerSoundloadRewind:
		if o.img.prefs.superchargerNotifications.Get().(bool) {
			icon = fmt.Sprintf("%c", fonts.Tape)
			secondaryIcon = fmt.Sprintf("%c", fonts.TapeRewind)
		}
	case notifications.NotifyPlusROMNetwork:
		if o.img.prefs.plusromNotifications.Get().(bool) {
			icon = fmt.Sprintf("%c", fonts.Wifi)
		}
	default:
		return
	}

	pos := imgui.Vec2{X: posMax.X, Y: posMin.Y}
	pos.X -= o.img.fonts.gopher2600IconsSize + overlayPadding
	if secondaryIcon != "" {
		pos.X -= o.img.fonts.largeFontAwesomeSize * 2
	}

	imgui.PushFont(o.img.fonts.gopher2600Icons)
	defer imgui.PopFont()

	imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: o.visibility, Y: o.visibility, Z: o.visibility, W: o.visibility})
	defer imgui.PopStyleColor()

	imgui.SetCursorScreenPos(pos)
	imgui.Text(icon)

	if secondaryIcon != "" {
		imgui.PushFont(o.img.fonts.largeFontAwesome)
		defer imgui.PopFont()

		imgui.SameLine()
		pos = imgui.CursorScreenPos()
		pos.Y += (o.img.fonts.gopher2600IconsSize - o.img.fonts.largeFontAwesomeSize) * 0.5

		imgui.SetCursorScreenPos(pos)
		imgui.Text(secondaryIcon)
	}
}

func (o *playscrOverlay) drawBottomLeft(posMin imgui.Vec2, posMax imgui.Vec2) {
	if !o.leftPortLatch.tick() {
		return
	}

	if !o.img.prefs.controllerNotifcations.Get().(bool) {
		return
	}

	pos := imgui.Vec2{X: posMin.X, Y: posMax.Y}
	pos.X += overlayPadding
	pos.Y -= o.img.fonts.gopher2600IconsSize + overlayPadding

	imgui.SetCursorScreenPos(pos)
	o.drawPeripheral(o.leftPort)
}

func (o *playscrOverlay) drawBottomRight(_ imgui.Vec2, posMax imgui.Vec2) {
	if !o.rightPortLatch.tick() {
		return
	}

	if !o.img.prefs.controllerNotifcations.Get().(bool) {
		return
	}

	pos := imgui.Vec2{X: posMax.X, Y: posMax.Y}
	pos.X -= o.img.fonts.gopher2600IconsSize + overlayPadding
	pos.Y -= o.img.fonts.gopher2600IconsSize + overlayPadding

	imgui.SetCursorScreenPos(pos)
	o.drawPeripheral(o.rightPort)
}

// drawPeripheral is used to draw the peripheral in the bottom left and bottom
// right corners of the overlay
func (o *playscrOverlay) drawPeripheral(peripID plugging.PeripheralID) {
	imgui.PushFont(o.img.fonts.gopher2600Icons)
	defer imgui.PopFont()

	imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: o.visibility, Y: o.visibility, Z: o.visibility, W: o.visibility})
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
	case plugging.PeriphKeyportari:
		imgui.Text(fmt.Sprintf("%c", fonts.Keyportari))
	}
}
