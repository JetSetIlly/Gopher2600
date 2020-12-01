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

	"github.com/inkyblackness/imgui-go/v2"
)

// the window menus grouped by type. the types are:.
const (
	windowMenuDebugger = "Debugger"
	windowMenuVCS      = "VCS"
	windowMenuCart     = "Cartridge"
	windowMenuOther    = "..."

	// additional window menus are grouped by cartridge type.
)

func (wm *manager) drawMenu() {
	if !imgui.BeginMainMenuBar() {
		return
	}

	// see commentary for screenPos in windowManager declaration
	wm.screenPos = imgui.WindowPos()

	if imgui.BeginMenu(windowMenuDebugger) {
		for _, id := range wm.menu[windowMenuDebugger] {
			drawMenuWindowEntry(wm.windows[id], id)
		}
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		if imgui.Selectable("  Quit") {
			wm.img.term.pushCommand("QUIT")
		}
		imgui.EndMenu()
	}

	// window menu
	if imgui.BeginMenu(windowMenuVCS) {
		for _, id := range wm.menu[windowMenuVCS] {
			drawMenuWindowEntry(wm.windows[id], id)
		}

		imgui.EndMenu()
	}

	// add cartridge specific menu if cartridge has a RAM bus or a debug bus.
	// note that debug bus windows need to have been added to the window menu
	// for the specific cartridge ID. see newWindowManager() function above
	cartSpecificMenu := wm.img.lz.Cart.HasRAMbus ||
		wm.img.lz.Cart.HasStaticBus ||
		wm.img.lz.Cart.HasCoProcBus

	if _, ok := wm.menu[wm.img.lz.Cart.ID]; ok {
		cartSpecificMenu = true
	}

	if cartSpecificMenu {
		if imgui.BeginMenu(fmt.Sprintf("Cart [%s]", wm.img.lz.Cart.ID)) {
			for _, id := range wm.menu[wm.img.lz.Cart.ID] {
				drawMenuWindowEntry(wm.windows[id], id)
			}

			if wm.img.lz.Cart.HasTapeBus {
				drawMenuWindowEntry(wm.windows[winCartTapeTitle], winCartTapeTitle)
			}

			if wm.img.lz.Cart.HasRAMbus {
				drawMenuWindowEntry(wm.windows[winCartRAMTitle], winCartRAMTitle)
			}

			if wm.img.lz.Cart.HasStaticBus {
				drawMenuWindowEntry(wm.windows[winCartStaticTitle], winCartStaticTitle)
			}

			imgui.EndMenu()
		}
	}

	// plusrom specific menus
	if wm.img.lz.Cart.IsPlusROM {
		if imgui.BeginMenu("PlusROM") {
			drawMenuWindowEntry(wm.windows[winPlusROMNetworkTitle], menuPlusROMNetworkTitle)
			drawMenuWindowEntry(wm.windows[winPlusROMPrefsTitle], menuPlusROMPrefsTitle)
			imgui.EndMenu()
		}
	}

	// add savekey specific menu
	if wm.img.lz.SaveKey.SaveKeyActive {
		if imgui.BeginMenu("SaveKey") {
			drawMenuWindowEntry(wm.windows[winSaveKeyI2CTitle], menuSaveKeyI2CTitle)
			drawMenuWindowEntry(wm.windows[winSaveKeyEEPROMTitle], menuSaveKeyEEPROMTitle)
			imgui.EndMenu()
		}
	}

	// filename in titlebar
	imgui.SameLineV(imgui.WindowWidth()-imguiGetFrameDim(wm.img.lz.Cart.Filename).X-20.0, 0.0)
	imgui.Text(wm.img.lz.Cart.Filename)

	imgui.EndMainMenuBar()
}

func drawMenuWindowEntry(w window, id string) {
	// decorate the menu entry with an "window open" indicator
	if w.isOpen() {
		// checkmark is unicode middle dot - code 00b7
		id = fmt.Sprintf("· %s", id)
	} else {
		id = fmt.Sprintf("  %s", id)
	}

	// window menu entries are toggleable
	if imgui.Selectable(id) {
		if w.isOpen() {
			w.setOpen(false)
		} else {
			w.setOpen(true)
		}
	}
}
