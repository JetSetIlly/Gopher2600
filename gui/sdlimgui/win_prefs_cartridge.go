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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
	"github.com/jetsetilly/imgui-go/v5"
)

func (win *winPrefs) drawCartridge() {
	imgui.Spacing()
	if imgui.CollapsingHeader("ARM") {
		win.drawARMTab()
	}
	imgui.Spacing()
	if imgui.CollapsingHeader("PlusROM") {
		win.drawPlusROMTab()
	}
	imgui.Spacing()
	if imgui.CollapsingHeader("SARA") {
		if !win.img.cache.VCS.Mem.Cart.HasSuperchip() {
			imgui.Spacing()
			imgui.Text("Current ROM does not have a SARA chip")
			imguiSeparator()
		}
		imgui.Spacing()
		win.drawSARA()
	}
}

func (win *winPrefs) drawSARA() bool {
	emulateSARA := win.img.dbg.VCS().Env.Prefs.Cartridge.EmulateSARA.Get().(bool)
	if imgui.Checkbox("Emulate Cycle Limitations", &emulateSARA) {
		win.img.dbg.VCS().Env.Prefs.Cartridge.EmulateSARA.Set(emulateSARA)
	}
	return emulateSARA
}

func (win *winPrefs) drawARMTab() {
	imgui.Spacing()

	if win.img.cache.VCS.Mem.Cart.GetCoProcBus() == nil {
		imgui.Text("Current ROM does not have an ARM coprocessor")
		imguiSeparator()
	}

	immediate := win.img.dbg.VCS().Env.Prefs.Cartridge.ARM.Immediate.Get().(bool)
	if imgui.Checkbox("Immediate ARM Execution", &immediate) {
		win.img.dbg.VCS().Env.Prefs.Cartridge.ARM.Immediate.Set(immediate)
	}
	win.img.imguiTooltipSimple("ARM program consumes no 6507 time")

	drawDisabled(immediate, func() {
		imgui.Spacing()

		var mamState string
		switch win.img.dbg.VCS().Env.Prefs.Cartridge.ARM.MAM.Get().(int) {
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
				win.img.dbg.VCS().Env.Prefs.Cartridge.ARM.MAM.Set(-1)
			}
			if imgui.Selectable("Disabled") {
				win.img.dbg.VCS().Env.Prefs.Cartridge.ARM.MAM.Set(0)
			}
			if imgui.Selectable("Partial") {
				win.img.dbg.VCS().Env.Prefs.Cartridge.ARM.MAM.Set(1)
			}
			if imgui.Selectable("Full") {
				win.img.dbg.VCS().Env.Prefs.Cartridge.ARM.MAM.Set(2)
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
		imgui.Separator()
		imgui.Spacing()

		clk := float32(win.img.dbg.VCS().Env.Prefs.Cartridge.ARM.Clock.Get().(float64))
		if imgui.SliderFloatV("Clock Speed", &clk, 50, 300, "%.0f Mhz", imgui.SliderFlagsNone) {
			win.img.dbg.VCS().Env.Prefs.Cartridge.ARM.Clock.Set(float64(clk))
		}

		imgui.Spacing()

		reg := float32(win.img.dbg.VCS().Env.Prefs.Cartridge.ARM.CycleRegulator.Get().(float64))
		if imgui.SliderFloatV("Cycle Regulator", &reg, 0.5, 2.0, "%.02f", imgui.SliderFlagsNone) {
			win.img.dbg.VCS().Env.Prefs.Cartridge.ARM.CycleRegulator.Set(float64(reg))
		}
		win.img.imguiTooltipSimple(`The cycle regulator is a way of adjusting the amount of
time each instruction in the ARM program takes`)
	})

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	abortOnMemoryFault := win.img.dbg.VCS().Env.Prefs.Cartridge.ARM.AbortOnMemoryFault.Get().(bool)
	if imgui.Checkbox("Abort on Memory Fault", &abortOnMemoryFault) {
		win.img.dbg.VCS().Env.Prefs.Cartridge.ARM.AbortOnMemoryFault.Set(abortOnMemoryFault)
	}

	undefinedSymbolWarning := win.img.dbg.VCS().Env.Prefs.Cartridge.ARM.UndefinedSymbolWarning.Get().(bool)
	if imgui.Checkbox("Undefined Symbols Warning", &undefinedSymbolWarning) {
		win.img.dbg.VCS().Env.Prefs.Cartridge.ARM.UndefinedSymbolWarning.Set(undefinedSymbolWarning)
	}
	win.img.imguiTooltipSimple(`It is possible to compile an ELF binary with undefined symbols.
This option presents causes a warning to appear when such a binary is loaded`)

	imgui.Spacing()
	if win.setDefaultButton("Set ARM Defaults") {
		win.img.dbg.PushFunction(win.img.dbg.VCS().Env.Prefs.Cartridge.ARM.SetDefaults)
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
