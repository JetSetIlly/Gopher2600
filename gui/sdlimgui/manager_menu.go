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

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
	"github.com/jetsetilly/imgui-go/v5"
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
	menuAtariVox
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

		imguiSeparator()

		if imgui.Selectable("  Reset") {
			wm.img.dbg.PushReset()
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

	// add atarivox savekey specific menu
	if wm.img.cache.VCS.GetAtariVox() != nil {
		if imgui.BeginMenu("AtariVox") {
			for _, m := range wm.menu[menuAtariVox] {
				wm.drawMenuEntry(m)
			}
			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()
			imgui.Text("SaveKey")
			for _, m := range wm.menu[menuSaveKey] {
				wm.drawMenuEntry(m)
			}
			imgui.EndMenu()
		}
	} else if wm.img.cache.VCS.GetSaveKey() != nil {
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

	// position cursor for drawing menu items from the right
	imgui.SetCursorScreenPos(imgui.Vec2{})
	imgui.SetCursorScreenPos(imgui.Vec2{
		X: imgui.ContentRegionAvail().X,
		Y: imgui.CursorScreenPos().Y,
	})

	spacing := imgui.CurrentStyle().ItemSpacing().X

	textFromRight := func(s string, separator bool, f func()) {
		// no separator but we still want to include the space. this helps any
		// additional entries after this one, particurlar buttons
		if !separator {
			s = " " + s
		}
		sz := imgui.CalcTextSize(s, false, -1)
		p := imgui.CursorScreenPos()
		p.X -= sz.X
		if separator {
			p.X -= spacing
		}
		imgui.SetCursorScreenPos(p)
		if separator {
			imgui.Separator()
		}
		imgui.Text(s)
		if f != nil {
			f()
		}
		if separator {
			p.X -= spacing
		}
		imgui.SetCursorScreenPos(p)
	}

	buttonFromRight := func(s string, leftclick func(), rightclick func()) {
		sz := imgui.CalcTextSize(s, false, -1)
		p := imgui.CursorScreenPos()
		p.X -= sz.X
		p.X -= spacing * 2
		imgui.SetCursorScreenPos(p)
		if imgui.Button(s) {
			if leftclick != nil {
				leftclick()
			}
		}
		if rightclick != nil && imgui.IsItemHovered() && imgui.IsMouseClicked(1) {
			rightclick()
		}
		p.X -= spacing
		imgui.SetCursorScreenPos(p)
	}

	// cartridge information
	textFromRight(fmt.Sprintf("%s %c", wm.img.cache.VCS.Mem.Cart.ShortName, fonts.Disk), true, nil)
	textFromRight(wm.img.cache.VCS.Mem.Cart.ID(), true, nil)
	banking := wm.img.cache.VCS.Mem.Cart.MappedBanks()
	if banking != "" {
		textFromRight(banking, true, nil)
	}

	// TV information
	frameInfo := wm.img.cache.TV.GetFrameInfo()
	if math.IsInf(float64(frameInfo.RefreshRate), 0) || frameInfo.RefreshRate > frameInfo.Spec.RefreshRate*2 {
		textFromRight("- Hz", true, nil)
	} else {
		textFromRight(fmt.Sprintf("%.2fHz", frameInfo.RefreshRate), true, nil)
	}

	// FPS information
	if wm.img.dbg.State() == govern.Running {
		actual, _ := wm.img.dbg.VCS().TV.GetActualFPS()
		if actual < 1.0 {
			textFromRight("< 1 fps", true, nil)
		} else if math.IsInf(float64(actual), 0) {
			textFromRight("- fps", true, nil)
		} else {
			textFromRight(fmt.Sprintf("%.1f fps", actual), true, nil)
		}
	} else {
		textFromRight("- fps", true, nil)
	}

	// tooltip control
	func() {
		wm.img.tooltipIndicator = false
		showTooltips := wm.img.prefs.showTooltips.Get().(bool)
		if !showTooltips {
			if !wm.img.tooltipIndicator {
				disabledAlpha := imgui.CurrentStyle().DisabledAlpha()
				imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
				defer imgui.PopStyleVar()
			}
		}
		textFromRight(string(fonts.SpeechBubble), false, func() {
			if imgui.IsItemClicked() {
				wm.img.prefs.showTooltips.Set(!showTooltips)
			}
		})
	}()

	// halt reason
	haltReason := wm.img.cache.Dbg.HaltReason
	if haltReason.Reason != "" {
		imgui.PushStyleColor(imgui.StyleColorButton, wm.img.cols.HaltReason)
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, wm.img.cols.HaltReasonHovered)
		imgui.PushStyleColor(imgui.StyleColorButtonActive, wm.img.cols.HaltReasonActive)
		buttonFromRight(fmt.Sprintf("Halt for %s", haltReason.Reason),
			func() {
				if coprocessor.CoProcYieldType(haltReason.Reason) == coprocessor.YieldMemoryFault {
					wm.debuggerWindows[winCoProcFaultsID].debuggerSetOpen(true)
				}
			},
			func() {
				wm.img.dbg.PushFunctionImmediate(wm.img.dbg.ClearHaltReason)
			},
		)
		imgui.PopStyleColorV(3)

		wm.img.imguiTooltip(func() {
			if haltReason.Detail != "" {
				imgui.Text(haltReason.Detail)
				imgui.Spacing()
				imgui.Separator()
				imgui.Spacing()
			}
			imgui.Textf("Frame: %d", haltReason.Coords.Frame)
			imgui.Textf("Scanline: %d", haltReason.Coords.Scanline)
			imgui.Textf("Clock: %d", haltReason.Coords.Clock)
		}, true)
	}
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
