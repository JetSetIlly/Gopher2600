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
	"strconv"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
	"github.com/jetsetilly/gopher2600/logger"
)

const winPrefsID = "Preferences"

type winPrefs struct {
	playmodeWin
	debuggerWin

	img *SdlImgui
}

func newWinPrefs(img *SdlImgui) (window, error) {
	win := &winPrefs{
		img: img,
	}

	return win, nil
}

func (win *winPrefs) init() {
}

func (win *winPrefs) id() string {
	return winPrefsID
}
func (win *winPrefs) playmodeDraw() bool {
	if !win.playmodeOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{100, 40}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	if imgui.BeginV(win.playmodeID(win.id()), &win.playmodeOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.playmodeWin.playmodeGeom.update()
	imgui.End()

	return true
}

func (win *winPrefs) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{29, 61}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerWin.debuggerGeom.update()
	imgui.End()

	return true
}

// the sub draw() functions may return a setDefaultPrefs instance. if an
// instance is returned then SetDefaults() will be used to draw a "Set
// Defaults" button
type setDefaultPrefs interface {
	SetDefaults()
}

func (win *winPrefs) draw() {
	var setDef setDefaultPrefs
	var setDefLabel = ""

	// tab-bar to switch between different "areas" of the TIA
	imgui.BeginTabBar("")
	if imgui.BeginTabItem("VCS") {
		win.drawVCS()
		imgui.EndTabItem()
	}

	if imgui.BeginTabItem("TV") {
		win.drawTV()
		imgui.EndTabItem()
	}

	if win.img.rnd.supportsCRT() {
		if imgui.BeginTabItem("CRT") {
			win.drawCRT()
			imgui.EndTabItem()
			setDef = win.img.crtPrefs
			setDefLabel = "CRT"
		}
	}

	if imgui.BeginTabItem("Playmode") {
		win.drawPlaymodeTab()
		imgui.EndTabItem()
	}

	if win.img.mode.Load().(govern.Mode) == govern.ModeDebugger {
		if imgui.BeginTabItem("Debugger") {
			win.drawDebuggerTab()
			imgui.EndTabItem()
		}
	}

	if imgui.BeginTabItem("Rewind") {
		win.drawRewindTab()
		imgui.EndTabItem()
		setDef = win.img.dbg.Rewind.Prefs
		setDefLabel = "Rewind"
	}

	if imgui.BeginTabItem("ARM") {
		win.drawARMTab()
		imgui.EndTabItem()
		setDef = win.img.dbg.VCS().Env.Prefs.ARM
		setDefLabel = "ARM"
	}

	if imgui.BeginTabItem("PlusROM") {
		win.drawPlusROMTab()
		imgui.EndTabItem()
	}

	if imgui.BeginTabItem("UI") {
		win.drawUITab()
		imgui.EndTabItem()
	}

	imgui.EndTabBar()

	imguiSeparator()
	win.drawDiskButtons()

	// draw "Set Defaults" button
	if setDef != nil {
		imgui.SameLine()
		if imgui.Button(fmt.Sprintf("Set %s Defaults", setDefLabel)) {
			// some preferences are sensitive to the goroutine SetDefaults() is
			// called within
			win.img.dbg.PushFunction(setDef.SetDefaults)
		}
	}
}

func (win *winPrefs) drawGlSwapInterval() {
	var glSwapInterval string

	const (
		descImmediate           = "Immediate updates"
		descWithVerticalRetrace = "Sync with vertical retrace"
		descAdaptive            = "Adaptive VSYNC"
		descTicker              = "Ticker"
	)

	switch win.img.prefs.glSwapInterval.Get().(int) {
	default:
		glSwapInterval = descImmediate
	case 1:
		glSwapInterval = descWithVerticalRetrace
	case -1:
		glSwapInterval = descAdaptive
	case 2:
		glSwapInterval = descTicker
	}

	if imgui.BeginCombo("Swap Interval", glSwapInterval) {
		if imgui.Selectable(descImmediate) {
			win.img.prefs.glSwapInterval.Set(syncImmediateUpdate)
		}
		if imgui.Selectable(descWithVerticalRetrace) {
			win.img.prefs.glSwapInterval.Set(syncWithVerticalRetrace)
		}
		if imgui.Selectable(descAdaptive) {
			win.img.prefs.glSwapInterval.Set(syncAdaptive)
		}
		if imgui.Selectable(descTicker) {
			win.img.prefs.glSwapInterval.Set(syncTicker)
		}
		imgui.EndCombo()
	}
}

func (win *winPrefs) drawPlaymodeTab() {
	imgui.Spacing()

	activePause := win.img.prefs.activePause.Get().(bool)
	if imgui.Checkbox("'Active' Pause Screen", &activePause) {
		win.img.prefs.activePause.Set(activePause)
	}
	win.img.imguiTooltipSimple(`An 'active' pause screen is one that tries to present
a television image that is sympathetic to the display kernel
of the ROM.`)

	imgui.Spacing()
	if imgui.CollapsingHeader("Notification Icons") {
		controllerNotifications := win.img.prefs.controllerNotifcations.Get().(bool)
		if imgui.Checkbox("Controller Changes", &controllerNotifications) {
			win.img.prefs.controllerNotifcations.Set(controllerNotifications)
		}

		plusromNotifications := win.img.prefs.plusromNotifications.Get().(bool)
		if imgui.Checkbox("PlusROM Network Activity", &plusromNotifications) {
			win.img.prefs.plusromNotifications.Set(plusromNotifications)
		}

		superchargerNotifications := win.img.prefs.superchargerNotifications.Get().(bool)
		if imgui.Checkbox("Supercharger Tape Motion", &superchargerNotifications) {
			win.img.prefs.superchargerNotifications.Set(superchargerNotifications)
		}

		audioMuteNotification := win.img.prefs.audioMuteNotification.Get().(bool)
		if imgui.Checkbox("Audio Mute Indicator", &audioMuteNotification) {
			win.img.prefs.audioMuteNotification.Set(audioMuteNotification)
		}

		visibility := float32(win.img.prefs.notificationVisibility.Get().(float64)) * 100
		if imgui.SliderFloatV("Visibility", &visibility, 0.0, 100.0, "%.0f%%", imgui.SliderFlagsNone) {
			win.img.prefs.notificationVisibility.Set(visibility / 100)
		}
	}

	imgui.Spacing()
	if imgui.CollapsingHeader("OpenGL Settings") {
		imgui.Spacing()
		win.drawGlSwapInterval()
	}

	imgui.Spacing()

	if imgui.CollapsingHeader("Frame Queue") {
		imgui.Spacing()

		// the values we show in the preferences window are the current values
		// as known by the screen type. we do no show the values in the
		// underlying preference type directly
		win.img.screen.crit.section.Lock()
		fpsCapped := win.img.screen.crit.fpsCapped
		frameQueueAuto := win.img.screen.crit.frameQueueAuto
		frameQueueLen := int32(win.img.screen.crit.frameQueueLen)
		win.img.screen.crit.section.Unlock()

		if !fpsCapped {
			imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
			imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
			defer imgui.PopItemFlag()
			defer imgui.PopStyleVar()
		}

		if imgui.Checkbox("Automatic Frame Queue Length", &frameQueueAuto) {
			win.img.prefs.frameQueueAuto.Set(frameQueueAuto)
		}

		imgui.Spacing()

		if frameQueueAuto {
			imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
			imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
			defer imgui.PopItemFlag()
			defer imgui.PopStyleVar()
		}

		if imgui.SliderInt("Frame Queue Length", &frameQueueLen, 1, maxFrameQueue) {
			win.img.prefs.frameQueue.Set(frameQueueLen)
		}
	}
}

func (win *winPrefs) drawDebuggerTab() {
	imgui.Spacing()

	audioMute := win.img.prefs.audioMuteDebugger.Get().(bool)
	if imgui.Checkbox("Audio Muted (in debugger)", &audioMute) {
		win.img.prefs.audioMuteDebugger.Set(audioMute)
	}

	termOnError := win.img.prefs.terminalOnError.Get().(bool)
	if imgui.Checkbox("Open Terminal on Error", &termOnError) {
		err := win.img.prefs.terminalOnError.Set(termOnError)
		if err != nil {
			logger.Logf("sdlimgui", "could not set preference value: %v", err)
		}
	}

	showTooltips := win.img.prefs.showTooltips.Get().(bool)
	if imgui.Checkbox("Show Tooltips", &showTooltips) {
		win.img.prefs.showTooltips.Set(showTooltips)
	}

	showTimelineThumbnail := win.img.prefs.showTimelineThumbnail.Get().(bool)
	if imgui.Checkbox("Show Thumbnail in Timeline", &showTimelineThumbnail) {
		win.img.prefs.showTimelineThumbnail.Set(showTimelineThumbnail)
	}

	imgui.Spacing()

	if imgui.CollapsingHeader("6507 Disassembly") {
		imgui.Spacing()
		usefxxmirror := win.img.dbg.Disasm.Prefs.FxxxMirror.Get().(bool)
		if imgui.Checkbox("Use Fxxx Mirror", &usefxxmirror) {
			win.img.dbg.Disasm.Prefs.FxxxMirror.Set(usefxxmirror)
		}

		usesymbols := win.img.dbg.Disasm.Prefs.Symbols.Get().(bool)
		if imgui.Checkbox("Use Symbols", &usesymbols) {
			win.img.dbg.Disasm.Prefs.Symbols.Set(usesymbols)

			// if disassembly has address labels then turning symbols off may alter
			// the vertical scrolling of the disassembly window.
			//
			// set focusOnAddr to true to force preference change to take effect
			win.img.wm.debuggerWindows[winDisasmID].(*winDisasm).focusOnAddr = true
		}

		colorDisasm := win.img.prefs.colorDisasm.Get().(bool)
		if imgui.Checkbox("Listing in Colour", &colorDisasm) {
			win.img.prefs.colorDisasm.Set(colorDisasm)
		}
	}

	imgui.Spacing()
	if imgui.CollapsingHeader("OpenGL Settings") {
		imgui.Spacing()
		win.drawGlSwapInterval()
	}
}

func (win *winPrefs) drawRewindTab() {
	imgui.Spacing()

	rewindMaxEntries := int32(win.img.dbg.Rewind.Prefs.MaxEntries.Get().(int))
	if imgui.SliderIntV("Max Entries##maxentries", &rewindMaxEntries, 10, 500, fmt.Sprintf("%d", rewindMaxEntries), imgui.SliderFlagsNone) {
		win.img.dbg.PushFunction(func() {
			win.img.dbg.Rewind.Prefs.MaxEntries.Set(rewindMaxEntries)
		})
	}

	imgui.Spacing()
	imguiIndentText("Changing these values will cause the")
	imguiIndentText("existing rewind history to be lost.")

	imgui.Spacing()
	imgui.Spacing()

	rewindFreq := int32(win.img.dbg.Rewind.Prefs.Freq.Get().(int))
	if imgui.SliderIntV("Frequency##freq", &rewindFreq, 1, 5, fmt.Sprintf("%d", rewindFreq), imgui.SliderFlagsNone) {
		win.img.dbg.PushFunction(func() {
			win.img.dbg.Rewind.Prefs.Freq.Set(rewindFreq)
		})
	}

	imgui.Spacing()
	imguiIndentText("Higher rewind frequencies may cause the")
	imguiIndentText("rewind controls to feel sluggish.")
}

func (win *winPrefs) drawVCS() {
	imgui.Spacing()

	if imgui.CollapsingHeaderV("Randomisation", imgui.TreeNodeFlagsDefaultOpen) {
		randState := win.img.dbg.VCS().Env.Prefs.RandomState.Get().(bool)
		if imgui.Checkbox("Random State (on startup)", &randState) {
			win.img.dbg.VCS().Env.Prefs.RandomState.Set(randState)
		}

		randPins := win.img.dbg.VCS().Env.Prefs.RandomPins.Get().(bool)
		if imgui.Checkbox("Random Pins", &randPins) {
			win.img.dbg.VCS().Env.Prefs.RandomPins.Set(randPins)
		}
	}

	imgui.Spacing()
	if imgui.CollapsingHeaderV("Audio", imgui.TreeNodeFlagsNone) {
		// enable options
		imgui.AlignTextToFramePadding()
		imgui.Text("Muted")
		imgui.SameLineV(0, 15)

		audioEnabledPlaymode := win.img.prefs.audioMutePlaymode.Get().(bool)
		if imgui.Checkbox("Playmode", &audioEnabledPlaymode) {
			win.img.prefs.audioMutePlaymode.Set(audioEnabledPlaymode)
		}

		imgui.SameLineV(0, 15)
		audioEnabledDebugger := win.img.prefs.audioMuteDebugger.Get().(bool)
		if imgui.Checkbox("Debugger", &audioEnabledDebugger) {
			win.img.prefs.audioMuteDebugger.Set(audioEnabledDebugger)
		}

		// stereo options
		stereo := win.img.audio.Prefs.Stereo.Get().(bool)
		if imgui.Checkbox("Stereo Sound", &stereo) {
			win.img.audio.Prefs.Stereo.Set(stereo)
		}

		if !stereo {
			imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
			imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
		}

		imgui.SameLineV(0, 15)
		discrete := win.img.audio.Prefs.Discrete.Get().(bool)
		if imgui.Checkbox("Discrete Channels", &discrete) {
			win.img.audio.Prefs.Discrete.Set(discrete)
		}

		if discrete {
			imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
			imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
		}

		// seperation values assume that there are three levels of effect
		// support by sdlaudio
		separation := int32(win.img.audio.Prefs.Separation.Get().(int))
		seperationLabel := ""
		switch separation {
		case 1:
			seperationLabel = "Narrow"
		case 2:
			seperationLabel = "Wide"
		case 3:
			seperationLabel = "Very Wide"
		}
		if imgui.SliderIntV("Separation", &separation, 1, 3, seperationLabel, 1.0) {
			win.img.audio.Prefs.Separation.Set(separation)
		}

		if discrete {
			imgui.PopStyleVar()
			imgui.PopItemFlag()
		}

		if !stereo {
			imgui.PopStyleVar()
			imgui.PopItemFlag()
		}
	}

	imgui.Spacing()
	if imgui.CollapsingHeaderV("TIA Revision", imgui.TreeNodeFlagsNone) {
		win.drawTIARev()
	}

	imgui.Spacing()
	if imgui.CollapsingHeader("AtariVox") {
		imgui.Spacing()
		imgui.Text("AtariVox output is currently only available")
		imgui.Text("via the Festival voice synthsizer. The path")
		imgui.Text("to the binary is specified below:")

		imgui.Spacing()
		binary := win.img.dbg.VCS().Env.Prefs.AtariVox.FestivalBinary.Get().(string)
		if imgui.InputTextV("##festivalbinary", &binary, imgui.InputTextFlagsEnterReturnsTrue, nil) {
			win.img.dbg.VCS().Env.Prefs.AtariVox.FestivalBinary.Set(binary)
			win.img.dbg.PushFunction(win.img.dbg.VCS().RIOT.Ports.RestartPeripherals)
		}

		imgui.Spacing()
		enabled := win.img.dbg.VCS().Env.Prefs.AtariVox.FestivalEnabled.Get().(bool)
		if imgui.Checkbox("Enable Festival Output", &enabled) {
			win.img.dbg.VCS().Env.Prefs.AtariVox.FestivalEnabled.Set(enabled)
			win.img.dbg.PushFunction(win.img.dbg.VCS().RIOT.Ports.RestartPeripherals)
		}

		var warning bool

		switch win.img.mode.Load().(govern.Mode) {
		case govern.ModePlay:
			warning = win.img.prefs.audioMutePlaymode.Get().(bool)
		case govern.ModeDebugger:
			warning = win.img.prefs.audioMuteDebugger.Get().(bool)
		}

		if warning && enabled {
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.Warning)
			imgui.AlignTextToFramePadding()
			imgui.Text(fmt.Sprintf(" %c", fonts.Warning))
			imgui.PopStyleColor()
			win.img.imguiTooltipSimple(`Emulation audio is currently muted. There will
be no AtariVox output even though the engine is
currently enabled.`)
		}
	}
}

func (win *winPrefs) drawARMTab() {
	imgui.Spacing()

	if win.img.cache.VCS.Mem.Cart.GetCoProcBus() == nil {
		imgui.Text("Current ROM does not have an ARM coprocessor")
		imguiSeparator()
	}

	immediate := win.img.dbg.VCS().Env.Prefs.ARM.Immediate.Get().(bool)
	if imgui.Checkbox("Immediate ARM Execution", &immediate) {
		win.img.dbg.VCS().Env.Prefs.ARM.Immediate.Set(immediate)
	}
	win.img.imguiTooltipSimple("ARM program consumes no 6507 time (like Stella)\nIf this option is set the other ARM settings are irrelevant")

	if immediate {
		imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
		imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
	}

	imgui.Spacing()

	var mamState string
	switch win.img.dbg.VCS().Env.Prefs.ARM.MAM.Get().(int) {
	case -1:
		mamState = "Driver"
	case 0:
		mamState = "Disabled"
	case 1:
		mamState = "Partial"
	case 2:
		mamState = "Full"
	}
	imgui.PushItemWidth(imguiGetFrameDim("Disabled").X + imgui.FrameHeight())
	if imgui.BeginComboV("Default MAM State##mam", mamState, imgui.ComboFlagsNone) {
		if imgui.Selectable("Driver") {
			win.img.dbg.VCS().Env.Prefs.ARM.MAM.Set(-1)
		}
		if imgui.Selectable("Disabled") {
			win.img.dbg.VCS().Env.Prefs.ARM.MAM.Set(0)
		}
		if imgui.Selectable("Partial") {
			win.img.dbg.VCS().Env.Prefs.ARM.MAM.Set(1)
		}
		if imgui.Selectable("Full") {
			win.img.dbg.VCS().Env.Prefs.ARM.MAM.Set(2)
		}
		imgui.EndCombo()
	}
	imgui.PopItemWidth()
	win.img.imguiTooltipSimple(`The MAM state at the start of the Thumb program.

For most purposes, this should be set to 'Driver'. This means that the emulated driver
for the cartridge mapper decides what the value should be.

If the 'Default MAM State' value is not set to 'Driver' then the Thumb program will be
prevented from changing the MAM state.

The MAM should almost never be disabled completely.`)

	imgui.Spacing()

	clk := float32(win.img.dbg.VCS().Env.Prefs.ARM.Clock.Get().(float64))
	if imgui.SliderFloatV("Clock Speed", &clk, 50, 300, "%.0f Mhz", imgui.SliderFlagsNone) {
		win.img.dbg.VCS().Env.Prefs.ARM.Clock.Set(float64(clk))
	}

	if immediate {
		imgui.PopStyleVar()
		imgui.PopItemFlag()
	}

	imgui.Spacing()

	abortOnMemoryFault := win.img.dbg.VCS().Env.Prefs.ARM.AbortOnMemoryFault.Get().(bool)
	if imgui.Checkbox("Abort on Memory Fault", &abortOnMemoryFault) {
		win.img.dbg.VCS().Env.Prefs.ARM.AbortOnMemoryFault.Set(abortOnMemoryFault)
	}
	win.img.imguiTooltipSimple(`Abort execution on a memory fault. For example when accessing
a memory address that does not exist.

Note that the program will always abort if the access is a PC fetch, even if this option is not set.

Illegal accesses will be logged even if program does not abort.`)

	misalignedAccessIsFault := win.img.dbg.VCS().Env.Prefs.ARM.MisalignedAccessIsFault.Get().(bool)
	if imgui.Checkbox("Treat Misaligned Accesses as Memory Faults", &misalignedAccessIsFault) {
		win.img.dbg.VCS().Env.Prefs.ARM.MisalignedAccessIsFault.Set(misalignedAccessIsFault)
	}
}

func (win *winPrefs) drawPlusROMTab() {
	imgui.Spacing()

	if _, ok := win.img.cache.VCS.Mem.Cart.GetContainer().(*plusrom.PlusROM); !ok {
		imgui.Text("Current ROM is not a PlusROM")
		imguiSeparator()
	}

	drawPlusROMNick(win.img)
}

func (win *winPrefs) drawUITab() {
	imgui.Spacing()

	if imgui.CollapsingHeaderV("Font Sizing and Spacing", imgui.TreeNodeFlagsDefaultOpen) {
		imgui.Spacing()

		var resetFonts bool

		const (
			minFontSize = 8
			maxFontSize = 30
		)

		// flags to be used in float slider
		sliderFlags := imgui.SliderFlagsAlwaysClamp | imgui.SliderFlagsNoInput

		// gui font
		guiSize := int32(win.img.prefs.guiFontSize.Get().(int))
		if imgui.SliderIntV("##guiFontSizeSlider", &guiSize, minFontSize, maxFontSize, "%dpt", sliderFlags) {
			win.img.prefs.guiFontSize.Set(guiSize)
		}
		if imgui.IsItemDeactivatedAfterEdit() {
			resetFonts = true
		}
		imgui.SameLineV(0, 5)

		guiSizeS := fmt.Sprintf("%d", guiSize)
		if imguiDecimalInput("GUI Font Size##guiFontSize", 3, &guiSizeS) {
			if sz, err := strconv.ParseInt(guiSizeS, 10, 32); err == nil {
				if sz >= minFontSize && sz <= maxFontSize {
					win.img.prefs.guiFontSize.Set(sz)
					resetFonts = true
				}
			}
		}

		imgui.Spacing()

		// terminal font
		terminalSize := int32(win.img.prefs.terminalFontSize.Get().(int))
		if imgui.SliderIntV("##terminalFontSizeSlider", &terminalSize, minFontSize, maxFontSize, "%dpt", sliderFlags) {
			win.img.prefs.terminalFontSize.Set(terminalSize)
		}
		if imgui.IsItemDeactivatedAfterEdit() {
			resetFonts = true
		}
		imgui.SameLineV(0, 5)

		terminalSizeS := fmt.Sprintf("%d", terminalSize)
		if imguiDecimalInput("Terminal Font Size##terminalFontSize", 3, &terminalSizeS) {
			if sz, err := strconv.ParseInt(terminalSizeS, 10, 32); err == nil {
				if sz >= minFontSize && sz <= maxFontSize {
					win.img.prefs.terminalFontSize.Set(sz)
					resetFonts = true
				}
			}
		}

		imgui.Spacing()

		// code font
		codeSize := int32(win.img.prefs.codeFontSize.Get().(int))
		if imgui.SliderIntV("##codeFontSizeSlider", &codeSize, minFontSize, maxFontSize, "%dpt", sliderFlags) {
			win.img.prefs.codeFontSize.Set(codeSize)
		}
		if imgui.IsItemDeactivatedAfterEdit() {
			resetFonts = true
		}
		imgui.SameLineV(0, 5)

		codeSizeS := fmt.Sprintf("%d", codeSize)
		if imguiDecimalInput("Code Font Size##codeFontSize", 3, &codeSizeS) {
			if sz, err := strconv.ParseInt(codeSizeS, 10, 32); err == nil {
				if sz >= minFontSize && sz <= maxFontSize {
					win.img.prefs.codeFontSize.Set(sz)
					resetFonts = true
				}
			}
		}

		imgui.Spacing()

		// code line spacing
		lineSpacing := int32(win.img.prefs.codeFontLineSpacing.Get().(int))
		if imgui.SliderInt("Line spacing in ARM Code window", &lineSpacing, 0, 5) {
			win.img.prefs.codeFontLineSpacing.Set(lineSpacing)
		}

		// reset fonts if prefs have changed
		if resetFonts {
			win.img.resetFonts = resetFontFrames
		}
	}
}

func (win *winPrefs) drawDiskButtons() {
	if imgui.Button("Save All") {
		err := win.img.prefs.save()
		if err != nil {
			logger.Logf("sdlimgui", "could not save (imgui debugger) preferences: %v", err)
		}
		err = win.img.crtPrefs.Save()
		if err != nil {
			logger.Logf("sdlimgui", "could not save (crt) preferences: %v", err)
		}
		err = win.img.audio.Prefs.Save()
		if err != nil {
			logger.Logf("sdlimgui", "could not save (sdlaudio) preferences: %v", err)
		}
		win.img.dbg.PushFunction(func() {
			err = win.img.dbg.VCS().Env.Prefs.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (vcs) preferences: %v", err)
			}
			err = win.img.dbg.VCS().Env.Prefs.ARM.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (arm) preferences: %v", err)
			}
			err = win.img.dbg.VCS().Env.Prefs.AtariVox.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (atarivox) preferences: %v", err)
			}
			err = win.img.dbg.VCS().Env.Prefs.PlusROM.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (plusrom) preferences: %v", err)
			}
			err = win.img.dbg.VCS().Env.Prefs.Revision.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (tia revisions) preferences: %v", err)
			}
			err = win.img.dbg.Rewind.Prefs.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (rewind) preferences: %v", err)
			}
			if win.img.mode.Load().(govern.Mode) == govern.ModeDebugger {
				err = win.img.dbg.Disasm.Prefs.Save()
				if err != nil {
					logger.Logf("sdlimgui", "could not save (disasm) preferences: %v", err)
				}
			}
		})
	}

	imgui.SameLine()
	if imgui.Button("Restore All") {
		err := win.img.prefs.load()
		if err != nil {
			logger.Logf("sdlimgui", "could not restore (imgui debugger) preferences: %v", err)
		}
		err = win.img.crtPrefs.Load()
		if err != nil {
			logger.Logf("sdlimgui", "could not restore (crt) preferences: %v", err)
		}
		err = win.img.audio.Prefs.Load()
		if err != nil {
			logger.Logf("sdlimgui", "could not restore (sdlaudio) preferences: %v", err)
		}
		win.img.dbg.PushFunction(func() {
			err = win.img.dbg.VCS().Env.Prefs.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (vcs) preferences: %v", err)
			}
			err = win.img.dbg.VCS().Env.Prefs.ARM.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (arm) preferences: %v", err)
			}
			err = win.img.dbg.VCS().Env.Prefs.AtariVox.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (atarivox) preferences: %v", err)
			}
			err = win.img.dbg.VCS().Env.Prefs.PlusROM.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (plusrom) preferences: %v", err)
			}
			err = win.img.dbg.VCS().Env.Prefs.Revision.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (tia revisions) preferences: %v", err)
			}
			err = win.img.dbg.Rewind.Prefs.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (rewind) preferences: %v", err)
			}
			if win.img.mode.Load().(govern.Mode) == govern.ModeDebugger {
				err = win.img.dbg.Disasm.Prefs.Load()
				if err != nil {
					logger.Logf("sdlimgui", "could not restore (disasm) preferences: %v", err)
				}
			}
		})

		win.img.resetFonts = resetFontFrames
	}
}
