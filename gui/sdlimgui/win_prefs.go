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
	"github.com/jetsetilly/gopher2600/logger"
)

const winPrefsID = "Preferences"

type winPrefs struct {
	img  *SdlImgui
	open bool
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

func (win *winPrefs) isOpen() bool {
	return win.open
}

func (win *winPrefs) setOpen(open bool) {
	win.open = open
}

// the sub draw() functions may return a setDefaultPrefs instance. if an
// instance is returned then SetDefaults() will be used to draw a "Set
// Defaults" button
type setDefaultPrefs interface {
	SetDefaults()
}

func (win *winPrefs) draw() {
	if !win.open {
		return
	}

	var setDef setDefaultPrefs
	var setDefLabel = ""

	if win.img.isPlaymode() {
		imgui.SetNextWindowPosV(imgui.Vec2{100, 40}, imgui.ConditionAppearing, imgui.Vec2{0, 0})
		imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsNoSavedSettings|imgui.WindowFlagsAlwaysAutoResize)
	} else {
		imgui.SetNextWindowPosV(imgui.Vec2{29, 61}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
		imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)
	}

	// tab-bar to switch between different "areas" of the TIA
	imgui.BeginTabBar("")
	if imgui.BeginTabItem("VCS") {
		win.drawVCS()
		imgui.EndTabItem()
	}

	if imgui.BeginTabItem("TIA Revisions") {
		win.drawTIARev()
		imgui.EndTabItem()
	}

	if imgui.BeginTabItem("CRT") {
		win.drawCRT()
		imgui.EndTabItem()
		setDef = win.img.crtPrefs
		setDefLabel = "CRT"
	}

	if imgui.BeginTabItem("Playmode") {
		win.drawPlaymode()
		imgui.EndTabItem()
	}

	if win.img.mode == emulation.ModeDebugger {
		if imgui.BeginTabItem("Debugger") {
			win.drawDebugger()
			imgui.EndTabItem()
		}
	}

	if imgui.BeginTabItem("Rewind") {
		win.drawRewind()
		imgui.EndTabItem()
		setDef = win.img.dbg.Rewind.Prefs
		setDefLabel = "Rewind"
	}

	if imgui.BeginTabItem("ARM") {
		win.drawARM()
		imgui.EndTabItem()
		setDef = win.img.vcs.Instance.Prefs.ARM
		setDefLabel = "ARM"
	}

	if imgui.BeginTabItem("PlusROM") {
		win.drawPlusROM()
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
			win.img.dbg.PushRawEvent(setDef.SetDefaults)
		}
	}

	imgui.End()
}

func (win *winPrefs) drawPlaymode() {
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
}

func (win *winPrefs) drawDebugger() {
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
		win.img.wm.windows[winDisasmID].(*winDisasm).focusOnAddr = true
	}

	audioEnabled := win.img.prefs.audioEnabled.Get().(bool)
	if imgui.Checkbox("Audio Enabled (in debugger)", &audioEnabled) {
		win.img.prefs.audioEnabled.Set(audioEnabled)
	}

	termOnError := win.img.prefs.openOnError.Get().(bool)
	if imgui.Checkbox("Open Terminal on Error", &termOnError) {
		err := win.img.prefs.openOnError.Set(termOnError)
		if err != nil {
			logger.Logf("sdlimgui", "could not set preference value: %v", err)
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
			if imgui.SliderInt("Line Spacing", &lineSpacing, 0, 5) {
				win.img.prefs.codeFontLineSpacing.Set(lineSpacing)
			}
		}
	}
}

func (win *winPrefs) drawRewind() {
	imgui.Spacing()

	rewindMaxEntries := int32(win.img.dbg.Rewind.Prefs.MaxEntries.Get().(int))
	if imgui.SliderIntV("Max Entries##maxentries", &rewindMaxEntries, 10, 500, fmt.Sprintf("%d", rewindMaxEntries), imgui.SliderFlagsNone) {
		win.img.dbg.PushRawEvent(func() {
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
		win.img.dbg.PushRawEvent(func() {
			win.img.dbg.Rewind.Prefs.Freq.Set(rewindFreq)
		})
	}

	imgui.Spacing()
	imguiIndentText("Higher rewind frequencies may cause the")
	imguiIndentText("rewind controls to feel sluggish.")
}

func (win *winPrefs) drawVCS() {
	imgui.Spacing()

	randState := win.img.vcs.Instance.Prefs.RandomState.Get().(bool)
	if imgui.Checkbox("Random State (on startup)", &randState) {
		win.img.vcs.Instance.Prefs.RandomState.Set(randState)
	}

	randPins := win.img.vcs.Instance.Prefs.RandomPins.Get().(bool)
	if imgui.Checkbox("Random Pins", &randPins) {
		win.img.vcs.Instance.Prefs.RandomPins.Set(randPins)
	}

	imguiSeparator()
	imgui.Text("Audio")
	imgui.Spacing()

	stereo := win.img.audio.Prefs.Stereo.Get().(bool)
	if imgui.Checkbox("Stereo Sound", &stereo) {
		win.img.audio.Prefs.Stereo.Set(stereo)
	}

	if !stereo {
		imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
		imgui.PushStyleVarFloat(imgui.StyleVarAlpha, 0.5)
	}

	separation := int32(win.img.audio.Prefs.Separation.Get().(int))

	label := ""
	switch separation {
	case 1:
		label = "Narrow"
	case 2:
		label = "Wide"
	case 3:
		label = "Discrete"
	}

	if imgui.SliderIntV("Separation", &separation, 1, 3, label, 1.0) {
		win.img.audio.Prefs.Separation.Set(separation)
	}

	if !stereo {
		imgui.PopStyleVar()
		imgui.PopItemFlag()
	}
}

func (win *winPrefs) drawARM() {
	imgui.Spacing()

	if !win.img.lz.Cart.HasCoProcBus {
		imgui.Text("Current ROM does not have an ARM coprocessor")
		imguiSeparator()
	}

	immediate := win.img.vcs.Instance.Prefs.ARM.Immediate.Get().(bool)
	if imgui.Checkbox("Immediate ARM Execution", &immediate) {
		win.img.vcs.Instance.Prefs.ARM.Immediate.Set(immediate)
	}
	imguiTooltipSimple("ARM program consumes no 6507 time (like Stella)\nIf this option is set the other ARM settings are irrelevant")

	if immediate {
		imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
		imgui.PushStyleVarFloat(imgui.StyleVarAlpha, 0.5)
	}

	imgui.Spacing()

	var mamState string
	switch win.img.vcs.Instance.Prefs.ARM.MAM.Get().(int) {
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
			win.img.vcs.Instance.Prefs.ARM.MAM.Set(-1)
		}
		if imgui.Selectable("Disabled") {
			win.img.vcs.Instance.Prefs.ARM.MAM.Set(0)
		}
		if imgui.Selectable("Partial") {
			win.img.vcs.Instance.Prefs.ARM.MAM.Set(1)
		}
		if imgui.Selectable("Full") {
			win.img.vcs.Instance.Prefs.ARM.MAM.Set(2)
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

	if immediate {
		imgui.PopStyleVar()
		imgui.PopItemFlag()
	}

	imgui.Spacing()

	abortOnIllegalMem := win.img.vcs.Instance.Prefs.ARM.AbortOnIllegalMem.Get().(bool)
	if imgui.Checkbox("Abort on Illegal Memory Access", &abortOnIllegalMem) {
		win.img.vcs.Instance.Prefs.ARM.AbortOnIllegalMem.Set(abortOnIllegalMem)
	}
	imguiTooltipSimple(`Abort thumb program on access to illegal memory. Note that the program
will always abort if the access is a PC fetch, even if this option is not set.

Illegal accesses will be logged in all instances.`)
}

func (win *winPrefs) drawPlusROM() {
	imgui.Spacing()

	if !win.img.lz.Cart.IsPlusROM {
		imgui.Text("Current ROM is not a PlusROM")
		imguiSeparator()
	}

	drawPlusROMNick(win.img)
}

func (win *winPrefs) drawDiskButtons() {
	if imgui.Button("Save All") {
		win.img.dbg.PushRawEvent(func() {
			err := win.img.prefs.save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (imgui debugger) preferences: %v", err)
			}
			err = win.img.audio.Prefs.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (sdlaudio) preferences: %v", err)
			}
			err = win.img.vcs.Instance.Prefs.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (vcs) preferences: %v", err)
			}
			err = win.img.vcs.Instance.Prefs.ARM.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (arm) preferences: %v", err)
			}
			err = win.img.vcs.Instance.Prefs.PlusROM.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (plusrom) preferences: %v", err)
			}
			err = win.img.vcs.Instance.Prefs.Revision.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (tia revisions) preferences: %v", err)
			}
			err = win.img.crtPrefs.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (crt) preferences: %v", err)
			}
			err = win.img.dbg.Rewind.Prefs.Save()
			if err != nil {
				logger.Logf("sdlimgui", "could not save (rewind) preferences: %v", err)
			}

			if win.img.mode == emulation.ModeDebugger {
				err = win.img.dbg.Disasm.Prefs.Save()
				if err != nil {
					logger.Logf("sdlimgui", "could not save (disasm) preferences: %v", err)
				}
			}
		})
	}

	imgui.SameLine()
	if imgui.Button("Restore All") {
		win.img.dbg.PushRawEvent(func() {
			err := win.img.prefs.load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (imgui debugger) preferences: %v", err)
			}
			err = win.img.audio.Prefs.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (sdlaudio) preferences: %v", err)
			}
			err = win.img.vcs.Instance.Prefs.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (vcs) preferences: %v", err)
			}
			err = win.img.vcs.Instance.Prefs.ARM.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (arm) preferences: %v", err)
			}
			err = win.img.vcs.Instance.Prefs.PlusROM.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (plusrom) preferences: %v", err)
			}
			err = win.img.vcs.Instance.Prefs.Revision.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (tia revisions) preferences: %v", err)
			}
			err = win.img.crtPrefs.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (crt) preferences: %v", err)
			}
			err = win.img.dbg.Rewind.Prefs.Load()
			if err != nil {
				logger.Logf("sdlimgui", "could not restore (rewind) preferences: %v", err)
			}

			if win.img.mode == emulation.ModeDebugger {
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
