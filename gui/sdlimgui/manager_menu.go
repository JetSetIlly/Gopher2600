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
	"math"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
)

// the window menus grouped by type. the types are:.
type menuGroup int

// list of valid menu groups.
const (
	menuDebugger menuGroup = iota
	menuVCS
	menuTools
	menuCart
	menuCoProc
	menuPlusROM
	menuSaveKey
	menuNone
)

// restrict menu entry for those cartridges with specific buses.
type menuRestrict int

// list of valid menu bus restrictions.
const (
	menuRestrictCoProc menuRestrict = iota
	menuRestrictRAM
	menuRestrictRegister
	menuRestrictStatic
	menuRestrictTape
)

type menuEntry struct {
	// the menu this menu entry appears under
	group menuGroup

	// restrictions on when menu entry can appear
	restrictBus    menuRestrict
	restrictMapper []string

	// the window thats referenced by the menu entry
	windowID string

	// the label that appears in the menu for this entry
	label string
}

func (wm *manager) drawMenu() {
	if !imgui.BeginMainMenuBar() {
		return
	}
	defer imgui.EndMainMenuBar()

	// debugger menu
	if imgui.BeginMenu("Debugger") {
		for _, m := range wm.menu[menuDebugger] {
			wm.drawMenuEntry(m)
		}

		// if imgui.Selectable("  Save ROM") {
		// 	wm.img.dbg.PushFunction(func() {
		// 		_, err := wm.img.dbg.VCS().(*hardware.VCS).Mem.Cart.ROMDump()
		// 		if err != nil {
		// 			logger.Log(logger.Allow, "save rom", err.Error())
		// 		}
		// 	})
		// }

		imguiSeparator()

		if imgui.Selectable("  Arrange Windows") {
			wm.arrangeBySize = 1
		}

		imguiSeparator()

		if imgui.Selectable("  Quit") {
			wm.img.term.pushCommand("QUIT")
		}

		imgui.EndMenu()
	}

	// vcs menu
	if imgui.BeginMenu("VCS") {
		for _, m := range wm.menu[menuVCS] {
			wm.drawMenuEntry(m)
		}
		imgui.EndMenu()
	}

	// tools menu
	if imgui.BeginMenu("Tools") {
		for _, m := range wm.menu[menuTools] {
			wm.drawMenuEntry(m)
		}
		imgui.EndMenu()
	}

	// cartridge menu. include test to see if menu should appear at all.
	if wm.img.cache.VCS.Mem.Cart.GetRAMbus() != nil || wm.img.cache.VCS.Mem.Cart.GetRegistersBus() != nil ||
		wm.img.cache.VCS.Mem.Cart.GetStaticBus() != nil || wm.img.cache.VCS.Mem.Cart.GetTapeBus() != nil {

		if imgui.BeginMenu("Cartridge") {
			for _, m := range wm.menu[menuCart] {
				wm.drawMenuEntry(m)
			}
			imgui.EndMenu()
		}
	}

	// coprocessor menu. include test to see if menu should appear at all.
	if coproc := wm.img.cache.VCS.Mem.Cart.GetCoProc(); coproc != nil {
		if imgui.BeginMenu(coproc.ProcessorID()) {
			for _, m := range wm.menu[menuCoProc] {
				wm.drawMenuEntry(m)
			}
			imgui.EndMenu()
		}
	}

	// plusrom specific menus
	if _, ok := wm.img.cache.VCS.Mem.Cart.GetContainer().(*plusrom.PlusROM); ok {
		if imgui.BeginMenu("PlusROM") {
			for _, m := range wm.menu[menuPlusROM] {
				wm.drawMenuEntry(m)
			}
			imgui.EndMenu()
		}
	}

	// add savekey specific menu
	if wm.img.cache.VCS.GetSaveKey() != nil {
		if imgui.BeginMenu("SaveKey") {
			for _, m := range wm.menu[menuSaveKey] {
				wm.drawMenuEntry(m)
			}
			imgui.EndMenu()
		}
	}

	// window search indicator
	if wm.searchActive {
		imgui.Text(" ")
		s := fmt.Sprintf("%c %s", fonts.MagnifyingGlass, wm.searchString)
		imguiColourButton(wm.img.cols.TitleBgActive, s, imguiGetFrameDim(s))
	}

	// cartridge info in menubar
	wdth := imgui.WindowWidth()
	wdth -= rightJustText(wdth, string(fonts.Disk), false)
	wdth -= rightJustText(wdth, wm.img.cache.VCS.Mem.Cart.ShortName, true)
	wdth -= rightJustText(wdth, wm.img.cache.VCS.Mem.Cart.ID(), true)
	wdth -= rightJustText(wdth, wm.img.cache.VCS.Mem.Cart.MappedBanks(), true)

	frameInfo := wm.img.cache.TV.GetFrameInfo()
	if math.IsInf(float64(frameInfo.RefreshRate), 0) || frameInfo.RefreshRate > frameInfo.Spec.RefreshRate*2 {
		wdth -= rightJustText(wdth, "- Hz", true)
		wm.img.imguiTooltip(func() { imgui.Text("TV refresh rate is indeterminate") }, true)
	} else {
		wdth -= rightJustText(wdth, fmt.Sprintf("%.2fHz", frameInfo.RefreshRate), true)
	}

	if wm.img.dbg.State() == govern.Running {
		actual, _ := wm.img.dbg.VCS().TV.GetActualFPS()
		req := wm.img.dbg.VCS().TV.GetReqFPS()
		if req < 1.0 {
			wdth -= rightJustText(wdth, "< 1 fps", true)
		} else if math.IsInf(float64(actual), 0) {
			wdth -= rightJustText(wdth, "- fps", true)
		} else {
			wdth -= rightJustText(wdth, fmt.Sprintf("%.1f fps", actual), true)
		}
	}

	wdth -= wm.drawTooltipIndicator(wdth)
	wm.img.tooltipIndicator = false
}

func (wm *manager) drawTooltipIndicator(wdth float32) float32 {
	s := fmt.Sprintf("%c", fonts.SpeechBubble)
	w := imguiGetFrameDim(s).X
	imgui.SameLineV(wdth-w, 0.0)
	showTooltips := wm.img.prefs.showTooltips.Get().(bool)
	if !showTooltips {
		if !wm.img.tooltipIndicator {
			imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
			defer imgui.PopStyleVar()
		}
	}
	imgui.BeginGroup()
	imgui.Text(s)
	imgui.EndGroup()
	if imgui.IsItemClicked() {
		wm.img.prefs.showTooltips.Set(!showTooltips)
	}
	return w
}

func rightJustText(width float32, text string, sep bool) float32 {
	w := imgui.CalcTextSize(text, false, 0.0).X +
		(imgui.CurrentStyle().FramePadding().X * 2) +
		(imgui.CurrentStyle().ItemInnerSpacing().X * 2)
	imgui.SameLineV(width-w, 0.0)
	if sep {
		imgui.Separator()
	}
	imgui.Text(text)
	return w
}

func (wm *manager) drawMenuEntry(m menuEntry) {
	// restriction bus
	switch m.restrictBus {
	case menuRestrictRAM:
		if wm.img.cache.VCS.Mem.Cart.GetRAMbus() == nil {
			return
		}
	case menuRestrictRegister:
		if wm.img.cache.VCS.Mem.Cart.GetRegistersBus() == nil {
			return
		}
	case menuRestrictStatic:
		if wm.img.cache.VCS.Mem.Cart.GetStaticBus() == nil {
			return
		}
	case menuRestrictTape:
		// additional test required for tape bus because of shortcomings in how
		// supercharger tapes/binary ROMs are implemented
		if bus := wm.img.cache.VCS.Mem.Cart.GetTapeBus(); bus == nil {
			return
		} else if ok, _ := bus.GetTapeState(); !ok {
			return
		}
	default:
		// no restrictions
	}

	// restrict bus
	restrict := len(m.restrictMapper) > 0
	if restrict {
		for _, r := range m.restrictMapper {
			if r == wm.img.cache.VCS.Mem.Cart.ID() {
				restrict = false
				break // for loop
			}
		}
		if restrict {
			return
		}
	}

	// the window that the menu entry refers to
	w := wm.debuggerWindows[m.windowID]

	// menu entry label. we'll decorate this with a "window open" indicator
	label := m.label
	if w.debuggerIsOpen() {
		// checkmark is unicode middle dot - code 00b7
		label = fmt.Sprintf("· %s", label)
	} else {
		label = fmt.Sprintf("  %s", label)
	}

	// window menu entries are toggleable
	if imgui.Selectable(label) {
		if w.debuggerIsOpen() {
			w.debuggerSetOpen(false)
		} else {
			w.debuggerSetOpen(true)
		}
	}
}
