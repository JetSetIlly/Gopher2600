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
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/harmony/arm7tdmi"
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

	win.drawARM()

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
		win.drawRewind()
	}

	imguiSeparator()
	win.drawDiskButtons()

	imgui.End()
}

// in this function we address vcs directly and not through the lazy system. it
// seems to be okay. acutal preference values are protected by mutexes in the
// prefs package so thats not a problem. the co-processor bus however can be
// contentious so we must be carefult during initialisation phase.
func (win *winPrefs) drawARM() {
	var hasARM bool

	// show ARM settings if we're in debugging mode or if there is an ARM coprocessor attached
	if win.img.isPlaymode() {
		// if emulation is "initialising" then return immediately
		//
		// !TODO: lazy system should be extended to work in playmode too. mainly to
		// help with situations like this. if we access the CoProcBus thought the
		// lazy system, we wouldn't need to check for initialising state.
		if win.img.state == gui.StateInitialising {
			return
		}

		bus := win.img.vcs.Mem.Cart.GetCoProcBus()
		hasARM = bus != nil && bus.CoProcID() == arm7tdmi.CoProcID

		if !hasARM {
			return
		}
	} else {
		hasARM = win.img.lz.CoProc.HasCoProcBus && win.img.lz.CoProc.ID == arm7tdmi.CoProcID
	}

	imguiSeparator()
	imgui.Text(arm7tdmi.CoProcID)
	if !hasARM {
		// not that the current cartridge does not have an ARM coprocessor
		imgui.SameLine()
		imgui.Text("(not in current cartridge)")
	}
	imgui.Spacing()

	immediate := win.img.vcs.Prefs.ARM.Immediate.Get().(bool)
	if imgui.Checkbox("Immediate ARM Execution", &immediate) {
		win.img.vcs.Prefs.ARM.Immediate.Set(immediate)
	}
	tooltipHover("ARM program consumes no 6507 time (like Stella)\nIf this option is set the other ARM settings are irrelevant")

	if immediate {
		imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
		imgui.PushStyleVarFloat(imgui.StyleVarAlpha, 0.5)
	}

	var mamState string
	switch win.img.vcs.Prefs.ARM.MAM.Get().(int) {
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
			win.img.vcs.Prefs.ARM.MAM.Set(-1)
		}
		if imgui.Selectable("Disabled") {
			win.img.vcs.Prefs.ARM.MAM.Set(0)
		}
		if imgui.Selectable("Partial") {
			win.img.vcs.Prefs.ARM.MAM.Set(1)
		}
		if imgui.Selectable("Full") {
			win.img.vcs.Prefs.ARM.MAM.Set(2)
		}
		imgui.EndCombo()
	}
	imgui.PopItemWidth()
	tooltipHover(`The MAM state at the start of the Thumb program.

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

	abortOnIllegalMem := win.img.vcs.Prefs.ARM.AbortOnIllegalMem.Get().(bool)
	if imgui.Checkbox("Abort on Illegal Memory Access", &abortOnIllegalMem) {
		win.img.vcs.Prefs.ARM.AbortOnIllegalMem.Set(abortOnIllegalMem)
	}
	tooltipHover(`Abort thumb program on access to illegal memory. Note that the program
will always abort if the access is a PC fetch, even if this option is not set.

Illegal accesses will be logged in all instances.`)
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
