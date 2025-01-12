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
	"github.com/jetsetilly/gopher2600/gui/sdlaudio"
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
	img     *SdlImgui
	playscr *playScr

	fps         string
	refreshRate string

	renderAlert int

	memStatsTicker *time.Ticker
	memStats       runtime.MemStats

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

func (ovly *playscrOverlay) set(v any, args ...any) {
	switch n := v.(type) {
	case plugging.PortID:
		switch n {
		case plugging.PortLeft:
			ovly.leftPort = args[0].(plugging.PeripheralID)
			ovly.leftPortLatch = overlayLatchShort
		case plugging.PortRight:
			ovly.rightPort = args[0].(plugging.PeripheralID)
			ovly.rightPortLatch = overlayLatchShort
		}
	case notifications.Notice:
		switch n {
		case notifications.NotifySuperchargerSoundloadStarted:
			ovly.cartridge = n
			ovly.cartridgeLatch = overlayLatchPinned
		case notifications.NotifySuperchargerSoundloadEnded:
			ovly.cartridge = n
			ovly.cartridgeLatch = overlayLatchShort
		case notifications.NotifySuperchargerSoundloadRewind:
			return

		case notifications.NotifyPlusROMNetwork:
			ovly.cartridge = n
			ovly.cartridgeLatch = overlayLatchShort

		case notifications.NotifyScreenshot:
			ovly.event = n
			ovly.eventLatch = overlayLatchShort

		default:
			return
		}
	}
}

func (ovly *playscrOverlay) draw() {
	imgui.PushStyleColor(imgui.StyleColorWindowBg, ovly.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorBorder, ovly.img.cols.Transparent)
	defer imgui.PopStyleColorV(2)

	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.Vec2{})
	defer imgui.PopStyleVarV(1)

	imgui.SetNextWindowPos(imgui.Vec2{X: 0, Y: 0})

	winw, winh := ovly.img.plt.windowSize()
	imgui.SetNextWindowSize(imgui.Vec2{X: winw, Y: winh})

	imgui.BeginV("##playscrOverlay", nil, imgui.WindowFlagsAlwaysAutoResize|
		imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|
		imgui.WindowFlagsNoDecoration|imgui.WindowFlagsNoSavedSettings|
		imgui.WindowFlagsNoBringToFrontOnFocus)
	defer imgui.End()

	ovly.visibility = float32(ovly.img.prefs.notificationVisibility.Get().(float64))

	ovly.drawTopLeft()
	ovly.drawTopRight()
	ovly.drawBottomLeft()
	ovly.drawBottomRight()
}

func (ovly *playscrOverlay) updateRefreshRate() {
	fps, refreshRate := ovly.img.dbg.VCS().TV.GetActualFPS()
	if fps == 0 {
		ovly.fps = "waiting"
	} else {
		ovly.fps = fmt.Sprintf("%03.2f fps", fps)
	}
	if refreshRate == 0 {
		ovly.refreshRate = "waiting"
	} else {
		ovly.refreshRate = fmt.Sprintf("%03.2fhz", refreshRate)
	}
}

// information in the top left corner of the overlay are about the emulation.
// eg. whether audio is mute, or the emulation is paused, etc. it is also used
// to display the FPS counter and other TV information
func (ovly *playscrOverlay) drawTopLeft() {
	pos := imgui.CursorScreenPos()
	pos.X += overlayPadding
	pos.Y += overlayPadding

	// by default only one icon is shown in the top left corner. however, if the
	// FPS overlay is being used we use the space to draw smaller icons
	var useIconQueue bool

	// draw FPS information if it's enabled
	if ovly.img.prefs.fpsDetail.Get().(bool) {
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

		ovly.updateRefreshRate()

		select {
		case <-ovly.memStatsTicker.C:
			runtime.ReadMemStats(&ovly.memStats)
		default:
		}

		imgui.Textf("Emulation: %s", ovly.fps)
		r := imgui.CurrentIO().Framerate()
		if r == 0.0 {
			imgui.Text("Rendering: waiting")
		} else {
			imgui.Textf("Rendering: %03.2f fps", r)
		}

		imguiSeparator()

		if coproc := ovly.img.cache.VCS.Mem.Cart.GetCoProc(); coproc != nil {
			clk := float32(ovly.img.dbg.VCS().Env.Prefs.ARM.Clock.Get().(float64))
			imgui.Text(fmt.Sprintf("%s Clock: %.0f Mhz", coproc.ProcessorID(), clk))
			imguiSeparator()
		}

		imgui.Text(fmt.Sprintf("%.1fx scaling", ovly.playscr.scaling))
		imgui.Text(fmt.Sprintf("%d total scanlines", ovly.playscr.scr.crit.frameInfo.TotalScanlines))

		imguiSeparator()

		// this construct (spacing followed by a same-line directive) is only
		// necessary so that the extreme left pixel of the VBLANKtop icon is not
		// chopped off. it's a very small detail but worth doing
		imgui.Spacing()
		imgui.SameLineV(0, 1)

		vblankBounds := fmt.Sprintf("%c %d  %c %d",
			fonts.VBLANKtop,
			ovly.playscr.scr.crit.frameInfo.VBLANKtop,
			fonts.VBLANKbottom,
			ovly.playscr.scr.crit.frameInfo.VBLANKbottom)
		vblankBounds = strings.ReplaceAll(vblankBounds, "-1", "-")
		imgui.Text(vblankBounds)
		if ovly.playscr.scr.crit.frameInfo.VBLANKunstable {
			imgui.SameLineV(0, 5)
			imgui.Text(string(fonts.Bug))
		}
		if ovly.playscr.scr.crit.frameInfo.AtariSafe() {
			imgui.SameLineV(0, 15)
			imgui.Text(string(fonts.VBLANKatari))
		}

		imgui.Spacing()
		if ovly.playscr.scr.crit.frameInfo.FromVSYNC {
			imgui.Text(fmt.Sprintf("VSYNC %d+%d", ovly.playscr.scr.crit.frameInfo.VSYNCscanline,
				ovly.playscr.scr.crit.frameInfo.VSYNCcount))
			if ovly.playscr.scr.crit.frameInfo.VSYNCunstable {
				imgui.SameLineV(0, 5)
				imgui.Text(string(fonts.Bug))
			}
		} else {
			imgui.Text(fmt.Sprintf("VSYNC %c", fonts.Bug))
		}

		imguiSeparator()
		imgui.Text(ovly.img.screen.crit.frameInfo.Spec.ID)

		imgui.SameLine()
		imgui.Text(ovly.refreshRate)

		if ovly.img.prefs.frameQueueMeterInOverlay.Get().(bool) {
			imguiSeparator()

			imgui.PushStyleColor(imgui.StyleColorText, ovly.img.cols.FrameQueueSlackActive)
			for _ = range ovly.playscr.scr.frameQueueSlack {
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
			}

			imgui.PushStyleColor(imgui.StyleColorText, ovly.img.cols.FrameQueueSlackInactive)
			for _ = range ovly.playscr.scr.crit.frameQueueLen - ovly.playscr.scr.frameQueueSlack {
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
			}
			imgui.Text("")

			imgui.PopStyleColorV(2)

			imgui.Spacing()
			imgui.Textf("%2.2fms/frame", float32(ovly.img.plt.renderAvgTime.Nanoseconds())/1000000)
			if ovly.img.plt.renderAlert {
				ovly.renderAlert = 60
			} else if ovly.renderAlert > 0 {
				ovly.renderAlert--
			}
			if ovly.renderAlert > 0 {
				imgui.SameLineV(0, 5)
				imgui.Text(string(fonts.RenderTime))
			}
		}

		if ovly.img.prefs.audioQueueMeterInOverlay.Get().(bool) {
			// draw separator if there is no frame queue meter
			if !ovly.img.prefs.frameQueueMeterInOverlay.Get().(bool) {
				imguiSeparator()
			} else {
				imgui.Spacing()
			}

			if ovly.img.audio.QueuedBytes == 0 {
				imgui.PushStyleColor(imgui.StyleColorText, ovly.img.cols.AudioQueueInactive)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.PopStyleColor()
			} else if ovly.img.audio.QueuedBytes < sdlaudio.QueueOkay {
				imgui.PushStyleColor(imgui.StyleColorText, ovly.img.cols.AudioQueueActive)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.PushStyleColor(imgui.StyleColorText, ovly.img.cols.AudioQueueInactive)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.PopStyleColorV(2)
			} else if ovly.img.audio.QueuedBytes < sdlaudio.QueueWarning {
				imgui.PushStyleColor(imgui.StyleColorText, ovly.img.cols.AudioQueueActive)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.PushStyleColor(imgui.StyleColorText, ovly.img.cols.AudioQueueInactive)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.PopStyleColorV(2)
			} else {
				imgui.PushStyleColor(imgui.StyleColorText, ovly.img.cols.AudioQueueActive)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.Text(string(fonts.MeterSegment))
				imgui.SameLineV(0, 0)
				imgui.PopStyleColor()
			}

			imgui.Spacing()
			if !ovly.img.prefs.audioMutePlaymode.Get().(bool) {
				imgui.Textf("%dkb audio queue", ovly.img.audio.QueuedBytes/1024)
			}
		}

		if ovly.img.prefs.memoryUsageInOverlay.Get().(bool) {
			imguiSeparator()
			imgui.Textf("Used = %v MB\n", ovly.memStats.Alloc/1048576)
			imgui.Textf("Reserved = %v MB\n", ovly.memStats.Sys/1048576)
			imgui.Textf("GC Sweeps = %v", ovly.memStats.NumGC)
			imgui.Textf("GC CPU %% = %.2f%%", ovly.memStats.GCCPUFraction*100)
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
		ovly.img.dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
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
		imgui.PushFont(ovly.img.fonts.veryLargeFontAwesome)
		defer imgui.PopFont()

		// add visibility adjustment if there is no FPS overlay
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: ovly.visibility, Y: ovly.visibility, Z: ovly.visibility, W: ovly.visibility})
		defer imgui.PopStyleColor()
	}

	// start a new icons queue
	ovly.iconQueue = ovly.iconQueue[:0]

	// mute is likely to be the icon visible the longest so has the lowest priority
	if ovly.img.prefs.audioMutePlaymode.Get().(bool) && ovly.img.prefs.audioMuteNotification.Get().(bool) {
		ovly.iconQueue = append(ovly.iconQueue, fonts.AudioMute)
	}

	// the real current state as set by the emulation is used to decide what
	// state to use for the overlay icon
	state := ovly.img.dbg.State()
	subState := ovly.img.dbg.SubState()

	switch state {
	case govern.Paused:
		// handling the pause state is the trickiest to get right. we want to
		// prioritise the pause icon in some cases but not in others
		switch ovly.state {
		case govern.Rewinding:
			// if the previous state was the rewinding state a pause icon will
			// show if the pause sub-state is not normal or if the
			// previous state latch has expired
			if subState != govern.Normal || ovly.stateLatch.expired() {
				ovly.state = state
				ovly.subState = subState
				ovly.stateLatch = overlayLatchPinned
			}
		default:
			ovly.state = state
			ovly.subState = subState
			ovly.stateLatch = overlayLatchPinned
		}
	case govern.Running:
		if state != ovly.state {
			ovly.state = state
			ovly.subState = subState
			ovly.stateLatch = overlayLatchShort
		}
	case govern.Rewinding:
		ovly.state = state
		ovly.subState = subState

		// refresh how the hold duration on every render frame that the
		// rewinding state is seen. this is so that the duration of the rewind
		// icon doesn't expire causing the pause icon to appear every so often
		//
		// (the way rewinding is implemented in the emulation means that the
		// rewinding state is interspersed very quickly with the paused state.
		// that works great for internal emulation purposes but requires careful
		// handling for UI purposes)
		ovly.stateLatch = overlayLatchBrief
	}

	// the state duration is ticked and the icon is shown unless the tick has
	// expired (returns false)
	if ovly.stateLatch.tick() {
		switch ovly.state {
		case govern.Paused:
			switch ovly.subState {
			case govern.PausedAtStart:
				ovly.iconQueue = append(ovly.iconQueue, fonts.EmulationPausedAtStart)
			case govern.PausedAtEnd:
				ovly.iconQueue = append(ovly.iconQueue, fonts.EmulationPausedAtEnd)
			default:
				ovly.iconQueue = append(ovly.iconQueue, fonts.EmulationPause)
			}
		case govern.Running:
			ovly.iconQueue = append(ovly.iconQueue, fonts.EmulationRun)
		case govern.Rewinding:
			switch ovly.subState {
			case govern.RewindingBackwards:
				ovly.iconQueue = append(ovly.iconQueue, fonts.EmulationRewindBack)
			case govern.RewindingForwards:
				ovly.iconQueue = append(ovly.iconQueue, fonts.EmulationRewindForward)
			default:
			}
		}
	}

	// events have the highest priority. we can think of these as user activated
	// events, such as the triggering of a screenshot. we therefore want to give
	// the user confirmation feedback immediately over other icons
	if ovly.eventLatch.tick() {
		switch ovly.event {
		case notifications.NotifyScreenshot:
			ovly.iconQueue = append(ovly.iconQueue, fonts.Camera)
		}
	}

	// draw only the last (ie. most important) icon unless the icon queue flag
	// has been set
	if !useIconQueue {
		if len(ovly.iconQueue) > 0 {
			imgui.Text(string(ovly.iconQueue[len(ovly.iconQueue)-1]))
		}
		return
	}

	// draw icons in order of priority
	for _, i := range ovly.iconQueue {
		imgui.Text(string(i))
		imgui.SameLine()
	}
	return
}

// information in the top right of the overlay is about the cartridge. ie.
// information from the cartridge about what is happening. for example,
// supercharger tape activity, or PlusROM network activity, etc.
func (ovly *playscrOverlay) drawTopRight() {
	if !ovly.cartridgeLatch.tick() {
		return
	}

	var icon string
	var secondaryIcon string

	switch ovly.cartridge {
	case notifications.NotifySuperchargerSoundloadStarted:
		if ovly.img.prefs.superchargerNotifications.Get().(bool) {
			icon = fmt.Sprintf("%c", fonts.Tape)
			secondaryIcon = fmt.Sprintf("%c", fonts.TapePlay)
		}
	case notifications.NotifySuperchargerSoundloadEnded:
		if ovly.img.prefs.superchargerNotifications.Get().(bool) {
			icon = fmt.Sprintf("%c", fonts.Tape)
			secondaryIcon = fmt.Sprintf("%c", fonts.TapeStop)
		}
	case notifications.NotifySuperchargerSoundloadRewind:
		if ovly.img.prefs.superchargerNotifications.Get().(bool) {
			icon = fmt.Sprintf("%c", fonts.Tape)
			secondaryIcon = fmt.Sprintf("%c", fonts.TapeRewind)
		}
	case notifications.NotifyPlusROMNetwork:
		if ovly.img.prefs.plusromNotifications.Get().(bool) {
			icon = fmt.Sprintf("%c", fonts.Wifi)
		}
	default:
		return
	}

	pos := imgui.WindowContentRegionMax()
	pos.X -= ovly.img.fonts.gopher2600IconsSize + overlayPadding
	pos.Y = 0
	if secondaryIcon != "" {
		pos.X -= ovly.img.fonts.largeFontAwesomeSize * 2
	}

	imgui.PushFont(ovly.img.fonts.gopher2600Icons)
	defer imgui.PopFont()

	imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: ovly.visibility, Y: ovly.visibility, Z: ovly.visibility, W: ovly.visibility})
	defer imgui.PopStyleColor()

	imgui.SetCursorScreenPos(pos)
	imgui.Text(icon)

	if secondaryIcon != "" {
		imgui.PushFont(ovly.img.fonts.largeFontAwesome)
		defer imgui.PopFont()

		imgui.SameLine()
		pos = imgui.CursorScreenPos()
		pos.Y += (ovly.img.fonts.gopher2600IconsSize - ovly.img.fonts.largeFontAwesomeSize) * 0.5

		imgui.SetCursorScreenPos(pos)
		imgui.Text(secondaryIcon)
	}
}

func (ovly *playscrOverlay) drawBottomLeft() {
	if !ovly.leftPortLatch.tick() {
		return
	}

	if !ovly.img.prefs.controllerNotifcations.Get().(bool) {
		return
	}

	pos := imgui.WindowContentRegionMax()
	pos.X = overlayPadding
	pos.Y -= ovly.img.fonts.gopher2600IconsSize + overlayPadding

	imgui.SetCursorScreenPos(pos)
	ovly.drawPeripheral(ovly.leftPort)
}

func (ovly *playscrOverlay) drawBottomRight() {
	if !ovly.rightPortLatch.tick() {
		return
	}

	if !ovly.img.prefs.controllerNotifcations.Get().(bool) {
		return
	}

	pos := imgui.WindowContentRegionMax()
	pos.X -= ovly.img.fonts.gopher2600IconsSize + overlayPadding
	pos.Y -= ovly.img.fonts.gopher2600IconsSize + overlayPadding

	imgui.SetCursorScreenPos(pos)
	ovly.drawPeripheral(ovly.rightPort)
}

// drawPeripheral is used to draw the peripheral in the bottom left and bottom
// right corners of the overlay
func (ovly *playscrOverlay) drawPeripheral(peripID plugging.PeripheralID) {
	imgui.PushFont(ovly.img.fonts.gopher2600Icons)
	defer imgui.PopFont()

	imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: ovly.visibility, Y: ovly.visibility, Z: ovly.visibility, W: ovly.visibility})
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
