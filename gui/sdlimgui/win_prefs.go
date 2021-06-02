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

	"github.com/inkyblackness/imgui-go/v4"
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

func (win *winPrefs) draw() {
	if !win.open {
		return
	}

	if win.img.isPlaymode() {
		imgui.SetNextWindowPosV(imgui.Vec2{25, 25}, imgui.ConditionAppearing, imgui.Vec2{0, 0})
		imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsNoSavedSettings|imgui.WindowFlagsAlwaysAutoResize)
	} else {
		imgui.SetNextWindowPosV(imgui.Vec2{29, 61}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
		imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)
	}

	imgui.Text("VCS")
	imgui.Spacing()

	randState := win.img.vcs.Prefs.RandomState.Get().(bool)
	if imgui.Checkbox("Random State (on startup)", &randState) {
		win.img.vcs.Prefs.RandomState.Set(randState)
	}

	randPins := win.img.vcs.Prefs.RandomPins.Get().(bool)
	if imgui.Checkbox("Random Pins", &randPins) {
		win.img.vcs.Prefs.RandomPins.Set(randPins)
	}

	imguiSeparator()
	imgui.Text("ARM7TDMI")
	imgui.Spacing()

	instantARM := win.img.vcs.Prefs.InstantARM.Get().(bool)
	if imgui.Checkbox("Instant ARM Execution", &instantARM) {
		win.img.vcs.Prefs.InstantARM.Set(instantARM)
	}
	win.drawHelp("ARM program consumes no 6507 time (like Stella)\nIf this option is set the other ARM settings are irrelevant")

	defaultMAM := win.img.vcs.Prefs.DefaultMAM.Get().(bool)
	if imgui.Checkbox("Default MAM Enable for Thumb Programs", &defaultMAM) {
		win.img.vcs.Prefs.DefaultMAM.Set(defaultMAM)
	}
	win.drawHelp("MAM will be enabled at beginning of every thumb program execution.\nRequired for new games like Gorf Arcade and Turbo")

	allowMAMfromThumb := win.img.vcs.Prefs.AllowMAMfromThumb.Get().(bool)
	if imgui.Checkbox("Allow MAM enable from Thumb", &allowMAMfromThumb) {
		win.img.vcs.Prefs.AllowMAMfromThumb.Set(allowMAMfromThumb)
	}
	win.drawHelp("MAM can be enabled/disabled by thumb program")

	if !win.img.isPlaymode() {
		imguiSeparator()
		imgui.Text("Debugger")
		imgui.Spacing()

		if imgui.Checkbox("Use Fxxx Mirror", &win.img.lz.Prefs.FxxxMirror) {
			win.img.term.pushCommand("PREFS TOGGLE FXXXMIRROR")
		}

		if imgui.Checkbox("Use Symbols", &win.img.lz.Prefs.Symbols) {
			win.img.term.pushCommand("PREFS TOGGLE SYMBOLS")

			// if disassembly has address labels then turning symbols off may alter
			// the vertical scrolling of the disassembly window.
			//
			// set focusOnAddr to true to force preference change to take effect
			win.img.wm.windows[winDisasmID].(*winDisasm).focusOnAddr = true
		}

		audioEnabled := win.img.prefs.audioEnabled.Get().(bool)
		if imgui.Checkbox("Audio Enabled", &audioEnabled) {
			win.img.prefs.audioEnabled.Set(audioEnabled)
		}

		termOnError := win.img.prefs.openOnError.Get().(bool)
		if imgui.Checkbox("Open Terminal on Error", &termOnError) {
			err := win.img.prefs.openOnError.Set(termOnError)
			if err != nil {
				logger.Logf("sdlimgui", "could not set preference value: %v", err)
			}
		}
		win.drawRewind()
	}

	imguiSeparator()
	win.drawDiskButtons()

	imgui.End()
}

func (win *winPrefs) drawHelp(text string) {
	if imgui.IsItemHovered() {
		imgui.BeginTooltip()
		defer imgui.EndTooltip()
		t := strings.Split(text, "\n")
		for i := range t {
			imgui.Text(t[i])
		}
	}
}

func (win *winPrefs) drawRewind() {
	imguiSeparator()
	imgui.Text("Rewind")
	imgui.Spacing()

	m := int32(win.img.lz.Prefs.RewindMaxEntries)
	if imgui.SliderIntV("Max Entries##maxentries", &m, 10, 100, fmt.Sprintf("%d", m), imgui.SliderFlagsNone) {
		win.img.term.pushCommand(fmt.Sprintf("PREFS REWIND MAX %d", m))
	}

	imgui.Spacing()
	imguiIndentText("Changing the max entries slider may cause")
	imguiIndentText("some of your rewind history to be lost.")

	imgui.Spacing()
	imgui.Spacing()

	f := int32(win.img.lz.Prefs.RewindFreq)
	if imgui.SliderIntV("Frequency##freq", &f, 1, 5, fmt.Sprintf("%d", f), imgui.SliderFlagsNone) {
		win.img.term.pushCommand(fmt.Sprintf("PREFS REWIND FREQ %d", f))
	}

	imgui.Spacing()
	imguiIndentText("Higher rewind frequencies may cause the")
	imguiIndentText("rewind controls to feel sluggish.")
}

func (win *winPrefs) drawDiskButtons() {
	if imgui.Button("Save") {
		err := win.img.prefs.save()
		if err != nil {
			logger.Logf("sdlimgui", "could not save (imgui debugger) preferences: %v", err)
		}
		err = win.img.vcs.Prefs.Save()
		if err != nil {
			logger.Logf("sdlimgui", "could not save (hardware) preferences: %v", err)
		}
		win.img.term.pushCommand("PREFS SAVE")
	}

	imgui.SameLine()
	if imgui.Button("Restore") {
		err := win.img.prefs.load()
		if err != nil {
			logger.Logf("sdlimgui", "could not restore (imgui debugger) preferences: %v", err)
		}
		err = win.img.vcs.Prefs.Load()
		if err != nil {
			logger.Logf("sdlimgui", "could not restore (hardware) preferences: %v", err)
		}
		win.img.term.pushCommand("PREFS LOAD")
	}
}
