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
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui/fonts"
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
func (win *winPrefs) playmodeDraw() {
	if !win.playmodeOpen {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{100, 40}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	if imgui.BeginV(win.playmodeID(win.id()), &win.playmodeOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.playmodeWin.playmodeGeom.update()
	imgui.End()
}

func (win *winPrefs) debuggerDraw() {
	if !win.debuggerOpen {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{29, 61}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerWin.debuggerGeom.update()
	imgui.End()
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

	if imgui.BeginTabItem("CRT") {
		win.drawCRT()
		imgui.EndTabItem()
		setDef = win.img.crtPrefs
		setDefLabel = "CRT"
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
		setDef = win.img.vcs.Env.Prefs.ARM
		setDefLabel = "ARM"
	}

	if imgui.BeginTabItem("PlusROM") {
		win.drawPlusROMTab()
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
	imguiTooltipSimple(`An 'active' pause screen is one that tries to present
a television image that is sympathetic to the display kernel
of the ROM.`)

	imgui.Spacing()
	imgui.Text("Notifications")
	imgui.Spacing()

	controllerNotifications := win.img.prefs.controllerNotifcations.Get().(bool)
	if imgui.Checkbox("Controller Change", &controllerNotifications) {
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

	imgui.Spacing()
	if imgui.CollapsingHeader("OpenGL Settings") {
		imgui.Spacing()
		win.drawGlSwapInterval()
	}

	imgui.Spacing()

	if imgui.CollapsingHeader("Frame Queue") {
		imgui.Spacing()

		func() {
			win.img.screen.crit.section.Lock()
			defer win.img.screen.crit.section.Unlock()

			if !win.img.screen.crit.fpsCapped {
				imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
				imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
				defer imgui.PopItemFlag()
				defer imgui.PopStyleVar()
			}

			frameQueueAuto := win.img.prefs.frameQueueAuto.Get().(bool)
			if imgui.Checkbox("Automatic Frame Queue Length", &frameQueueAuto) {
				win.img.prefs.frameQueueAuto.Set(frameQueueAuto)
				win.img.screen.updateFrameQueue()
			}

			imgui.Spacing()

			if frameQueueAuto {
				imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
				imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
				defer imgui.PopItemFlag()
				defer imgui.PopStyleVar()
			}

			frameQueue := int32(win.img.screen.crit.frameQueueLen)
			if imgui.SliderInt("Frame Queue Length", &frameQueue, 1, maxFrameQueue) {
				win.img.prefs.frameQueue.Set(frameQueue)
				win.img.screen.updateFrameQueue()
			}
		}()
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

	// font preferences for when compiled with freetype font rendering
	if win.img.glsl.fonts.isFreeType() {
		imgui.Spacing()

		if imgui.CollapsingHeader("Font Sizes") {
			imgui.Spacing()

			guiSize := win.img.prefs.guiFont.Get().(float64)
			if imgui.BeginCombo("GUI", fmt.Sprintf("%.01f", guiSize)) {
				if imgui.Selectable("12.0") {
					win.img.prefs.guiFont.Set(12.0)
					win.img.resetFonts = resetFontFrames
				}
				if imgui.Selectable("13.0") {
					win.img.prefs.guiFont.Set(13.0)
					win.img.resetFonts = resetFontFrames
				}
				if imgui.Selectable("14.0") {
					win.img.prefs.guiFont.Set(14.0)
					win.img.resetFonts = resetFontFrames
				}
				if imgui.Selectable("15.0") {
					win.img.prefs.guiFont.Set(15.0)
					win.img.resetFonts = resetFontFrames
				}
				if imgui.Selectable("16.0") {
					win.img.prefs.guiFont.Set(16.0)
					win.img.resetFonts = resetFontFrames
				}
				imgui.EndCombo()
			}

			imgui.Spacing()

			codeSize := win.img.prefs.codeFont.Get().(float64)
			if imgui.BeginCombo("Code", fmt.Sprintf("%.01f", codeSize)) {
				if imgui.Selectable("13.0") {
					win.img.prefs.codeFont.Set(13.0)
					win.img.resetFonts = resetFontFrames
				}
				if imgui.Selectable("14.0") {
					win.img.prefs.codeFont.Set(14.0)
					win.img.resetFonts = resetFontFrames
				}
				if imgui.Selectable("15.0") {
					win.img.prefs.codeFont.Set(15.0)
					win.img.resetFonts = resetFontFrames
				}
				if imgui.Selectable("16.0") {
					win.img.prefs.codeFont.Set(16.0)
					win.img.resetFonts = resetFontFrames
				}
				if imgui.Selectable("17.0") {
					win.img.prefs.codeFont.Set(17.0)
					win.img.resetFonts = resetFontFrames
				}
				imgui.EndCombo()
			}

			imgui.Spacing()

			lineSpacing := int32(win.img.prefs.codeFontLineSpacing.Get().(int))
			if imgui.SliderInt("Line Spacing (code)", &lineSpacing, 0, 5) {
				win.img.prefs.codeFontLineSpacing.Set(lineSpacing)
			}
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

	imgui.Spacing()
	if imgui.CollapsingHeaderV("Randomisation", imgui.TreeNodeFlagsDefaultOpen) {
		randState := win.img.vcs.Env.Prefs.RandomState.Get().(bool)
		if imgui.Checkbox("Random State (on startup)", &randState) {
			win.img.vcs.Env.Prefs.RandomState.Set(randState)
		}

		randPins := win.img.vcs.Env.Prefs.RandomPins.Get().(bool)
		if imgui.Checkbox("Random Pins", &randPins) {
			win.img.vcs.Env.Prefs.RandomPins.Set(randPins)
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
		binary := win.img.vcs.Env.Prefs.AtariVox.FestivalBinary.Get().(string)
		if imgui.InputTextV("##festivalbinary", &binary, imgui.InputTextFlagsEnterReturnsTrue, nil) {
			win.img.vcs.Env.Prefs.AtariVox.FestivalBinary.Set(binary)
			win.img.dbg.PushFunction(win.img.vcs.RIOT.Ports.RestartPeripherals)
		}

		imgui.Spacing()
		enabled := win.img.vcs.Env.Prefs.AtariVox.FestivalEnabled.Get().(bool)
		if imgui.Checkbox("Enable Festival Output", &enabled) {
			win.img.vcs.Env.Prefs.AtariVox.FestivalEnabled.Set(enabled)
			win.img.dbg.PushFunction(win.img.vcs.RIOT.Ports.RestartPeripherals)
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
			imguiTooltipSimple(`Emulation audio is currently muted. There will
be no AtariVox output even though the engine is
currently enabled.`)
		}
	}
}

func (win *winPrefs) drawARMTab() {
	imgui.Spacing()

	if !win.img.lz.Cart.HasCoProcBus {
		imgui.Text("Current ROM does not have an ARM coprocessor")
		imguiSeparator()
	}

	immediate := win.img.vcs.Env.Prefs.ARM.Immediate.Get().(bool)
	if imgui.Checkbox("Immediate ARM Execution", &immediate) {
		win.img.vcs.Env.Prefs.ARM.Immediate.Set(immediate)
	}
	imguiTooltipSimple("ARM program consumes no 6507 time (like Stella)\nIf this option is set the other ARM settings are irrelevant")

	if immediate {
		imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
		imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
	}

	imgui.Spacing()

	var mamState string
	switch win.img.vcs.Env.Prefs.ARM.MAM.Get().(int) {
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
			win.img.vcs.Env.Prefs.ARM.MAM.Set(-1)
		}
		if imgui.Selectable("Disabled") {
			win.img.vcs.Env.Prefs.ARM.MAM.Set(0)
		}
		if imgui.Selectable("Partial") {
			win.img.vcs.Env.Prefs.ARM.MAM.Set(1)
		}
		if imgui.Selectable("Full") {
			win.img.vcs.Env.Prefs.ARM.MAM.Set(2)
		}
		imgui.EndCombo()
	}
	imgui.PopItemWidth()
	imguiTooltipSimple(`The MAM state at the start of the Thumb program.

For most purposes, this should be set to 'Driver'. This means that the emulated driver
for the cartridge mapper decides what the value should be.

If the 'Default MAM State' value is not set to 'Driver' then the Thumb program will be
prevented from changing the MAM state.

The MAM should almost never be disabled completely.`)

	imgui.Spacing()

	clk := float32(win.img.vcs.Env.Prefs.ARM.Clock.Get().(float64))
	if imgui.SliderFloatV("Clock Speed", &clk, 50, 80, "%.0f Mhz", imgui.SliderFlagsNone) {
		win.img.vcs.Env.Prefs.ARM.Clock.Set(float64(clk))
	}

	if immediate {
		imgui.PopStyleVar()
		imgui.PopItemFlag()
	}

	imgui.Spacing()

	if imgui.CollapsingHeader("Abort Conditions") {
		imgui.Spacing()

		abortOnIllegalMem := win.img.vcs.Env.Prefs.ARM.AbortOnIllegalMem.Get().(bool)
		if imgui.Checkbox("Illegal Memory Access", &abortOnIllegalMem) {
			win.img.vcs.Env.Prefs.ARM.AbortOnIllegalMem.Set(abortOnIllegalMem)
		}
		imguiTooltipSimple(`Abort thumb program on access to illegal memory. Note that the program
will always abort if the access is a PC fetch, even if this option is not set.

Illegal accesses will be logged even if program does not abort.`)

		abortOnStackCollision := win.img.vcs.Env.Prefs.ARM.AbortOnStackCollision.Get().(bool)
		if imgui.Checkbox("Stack Collision", &abortOnStackCollision) {
			win.img.vcs.Env.Prefs.ARM.AbortOnStackCollision.Set(abortOnStackCollision)
		}
		imguiTooltipSimple(`Abort thumb program if stack pointer overlaps the highest address
occupied by a variable in the program.

Only available when DWARF data is available for the program.

Stack collisions will be logged even if program does not abort.`)
	}

}

func (win *winPrefs) drawPlusROMTab() {
	imgui.Spacing()

	if !win.img.lz.Cart.IsPlusROM {
		imgui.Text("Current ROM is not a PlusROM")
		imguiSeparator()
	}

	drawPlusROMNick(win.img)
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
			err = win.img.vcs.Env.Prefs.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (vcs) preferences: %v", err)
			}
			err = win.img.vcs.Env.Prefs.ARM.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (arm) preferences: %v", err)
			}
			err = win.img.vcs.Env.Prefs.AtariVox.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (atarivox) preferences: %v", err)
			}
			err = win.img.vcs.Env.Prefs.PlusROM.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (plusrom) preferences: %v", err)
			}
			err = win.img.vcs.Env.Prefs.Revision.Save()
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
			err = win.img.vcs.Env.Prefs.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (vcs) preferences: %v", err)
			}
			err = win.img.vcs.Env.Prefs.ARM.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (arm) preferences: %v", err)
			}
			err = win.img.vcs.Env.Prefs.AtariVox.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (atarivox) preferences: %v", err)
			}
			err = win.img.vcs.Env.Prefs.PlusROM.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (plusrom) preferences: %v", err)
			}
			err = win.img.vcs.Env.Prefs.Revision.Load()
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

		if win.img.glsl.fonts.isFreeType() {
			win.img.resetFonts = resetFontFrames
		}
	}
}
