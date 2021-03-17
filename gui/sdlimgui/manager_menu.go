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
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/fonts"
)

// the window menus grouped by type. the types are:.
type menuGroup int

// list of valid menu groups.
const (
	menuDebugger menuGroup = iota
	menuVCS
	menuCart
	menuCoProc
	menuPlusROM
	menuSaveKey
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

	// cartridge menu. include test to see if menu should appear at all.
	if wm.img.lz.Cart.HasRAMbus || wm.img.lz.Cart.HasRegistersBus || wm.img.lz.Cart.HasStaticBus || wm.img.lz.Cart.HasTapeBus {
		if imgui.BeginMenu("Cartridge") {
			for _, m := range wm.menu[menuCart] {
				wm.drawMenuEntry(m)
			}
			imgui.EndMenu()
		}
	}

	// coprocessor menu. include test to see if menu should appear at all.
	if wm.img.lz.CoProc.HasCoProcBus {
		if imgui.BeginMenu(wm.img.lz.CoProc.ID) {
			for _, m := range wm.menu[menuCoProc] {
				wm.drawMenuEntry(m)
			}
			imgui.EndMenu()
		}
	}

	// plusrom specific menus
	if wm.img.lz.Cart.IsPlusROM {
		if imgui.BeginMenu("PlusROM") {
			for _, m := range wm.menu[menuPlusROM] {
				wm.drawMenuEntry(m)
			}
			imgui.EndMenu()
		}
	}

	// add savekey specific menu
	if wm.img.lz.SaveKey.SaveKeyActive {
		if imgui.BeginMenu("SaveKey") {
			for _, m := range wm.menu[menuSaveKey] {
				wm.drawMenuEntry(m)
			}
			imgui.EndMenu()
		}
	}

	// cartridge info in menubar
	wdth := imgui.WindowWidth()
	wdth -= rightJustText(wdth, string(fonts.Disk), false)
	wdth -= rightJustText(wdth, wm.img.lz.Cart.Filename, true)
	wdth -= rightJustText(wdth, wm.img.lz.Cart.ID, true)
	wdth -= rightJustText(wdth, wm.img.lz.Cart.Mapping, true)

	if wm.img.state == gui.StateRunning {
		if wm.img.lz.TV.ReqFPS < 1.0 {
			rightJustText(wdth, "< 1 fps", true)
		} else {
			rightJustText(wdth, fmt.Sprintf("%.1f fps", wm.img.lz.TV.ActualFPS), true)
		}
	}
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
		if !wm.img.lz.Cart.HasRAMbus {
			return
		}
	case menuRestrictRegister:
		if !wm.img.lz.Cart.HasRegistersBus {
			return
		}
	case menuRestrictStatic:
		if !wm.img.lz.Cart.HasStaticBus {
			return
		}
	case menuRestrictTape:
		if !wm.img.lz.Cart.HasTapeBus {
			return
		}
	default:
		// no restrictions
	}

	// restrict bus
	restrict := len(m.restrictMapper) > 0
	if restrict {
		for _, r := range m.restrictMapper {
			if r == wm.img.lz.Cart.ID {
				restrict = false
				break // for loop
			}
		}
		if restrict {
			return
		}
	}

	// the window that the menu entry refers to
	w := wm.windows[m.windowID]

	// menu entry label. we'll decorate this with an "window open" indicator
	label := m.label
	if w.isOpen() {
		// checkmark is unicode middle dot - code 00b7
		label = fmt.Sprintf("· %s", label)
	} else {
		label = fmt.Sprintf("  %s", label)
	}

	// window menu entries are toggleable
	if imgui.Selectable(label) {
		if w.isOpen() {
			w.setOpen(false)
		} else {
			w.setOpen(true)
		}
	}
}
