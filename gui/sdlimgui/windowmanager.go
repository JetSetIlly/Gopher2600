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
	"sort"

	"github.com/inkyblackness/imgui-go/v2"
)

// managedWindow conceptualises the functions required by a window such that
// it can be managed by the windowManager.
type managedWindow interface {
	init()
	id() string
	destroy()
	draw()
	isOpen() bool
	setOpen(bool)
}

// this file contains some useful emeddable types for implementations of the
// managedWindow interface

// windowManagement can be embedded into a real window struct for
// basic window management functionality. it partially implements the
// managedWindow interface.
type windowManagement struct {
	// prefer use of isOpen()/setOpen() instead of accessing the open field
	// directly
	open bool
}

func (wm *windowManagement) isOpen() bool {
	return wm.open
}

func (wm *windowManagement) setOpen(open bool) {
	wm.open = open
}

// windowManager handles windows and menus in the system.
type windowManager struct {
	img *SdlImgui

	// has the window manager gone through the initialisation process
	hasInitialised bool

	// the collection of managed windows in the system, indexed by window title
	windows map[string]managedWindow

	// windows can be open and closed through the menu bar. they are grouped
	// according to type using the windowMenu constants defined below.
	windowMenu map[string][]string

	// some windows need to be referenced elsewhere
	term    *winTerm
	dbgScr  *winDbgScr
	playScr *winPlayScr

	// the position of the screen on the current display. the SDL function
	// Window.GetPosition() is unsuitable for use in conjunction with imgui
	// because it considers screen space across all display devices, imgui does
	// not.
	//
	// screenPos is an alternative to the SDL GetPosition() function. we get
	// the value by asking for the screenPos of the main menu. because the main
	// menu is always in the very top-left corner of the window it is a good
	// proxy value
	screenPos imgui.Vec2
}

// the window menus grouped by type. the types are:.
const (
	windowMenuDebugger = "Debugger"
	windowMenuVCS      = "VCS"
	windowMenuCart     = "Cartridge"
	windowMenuOther    = "..."

	// additional window menus are grouped by cartridge type.
)

func newWindowManager(img *SdlImgui) (*windowManager, error) {
	wm := &windowManager{
		img:        img,
		windows:    make(map[string]managedWindow),
		windowMenu: make(map[string][]string),
	}

	// creation function for all managed windows
	addWindow := func(create func(img *SdlImgui) (managedWindow, error), open bool, group string) error {
		w, err := create(img)
		if err != nil {
			return err
		}

		wm.windows[w.id()] = w
		wm.windowMenu[group] = append(wm.windowMenu[group], w.id())
		w.setOpen(open)

		return nil
	}

	// windows called from project menu
	if err := addWindow(newFileSelector, false, windowMenuDebugger); err != nil {
		return nil, err
	}
	if err := addWindow(newWinPrefs, false, windowMenuDebugger); err != nil {
		return nil, err
	}
	if err := addWindow(newWinTerm, false, windowMenuDebugger); err != nil {
		return nil, err
	}
	if err := addWindow(newWinLog, false, windowMenuDebugger); err != nil {
		return nil, err
	}

	// windows that appear in the "windows" menu
	if err := addWindow(newWinControl, true, windowMenuVCS); err != nil {
		return nil, err
	}
	if err := addWindow(newWinCPU, true, windowMenuVCS); err != nil {
		return nil, err
	}
	if err := addWindow(newWinRAM, true, windowMenuVCS); err != nil {
		return nil, err
	}
	if err := addWindow(newWinTIA, true, windowMenuVCS); err != nil {
		return nil, err
	}
	if err := addWindow(newWinTimer, true, windowMenuVCS); err != nil {
		return nil, err
	}
	if err := addWindow(newWinDisasm, true, windowMenuVCS); err != nil {
		return nil, err
	}
	if err := addWindow(newWinAudio, true, windowMenuVCS); err != nil {
		return nil, err
	}
	if err := addWindow(newWinDbgScr, true, windowMenuVCS); err != nil {
		return nil, err
	}
	if err := addWindow(newWinControllers, false, windowMenuVCS); err != nil {
		return nil, err
	}
	if err := addWindow(newWinCollisions, false, windowMenuVCS); err != nil {
		return nil, err
	}
	if err := addWindow(newWinChipRegisters, false, windowMenuVCS); err != nil {
		return nil, err
	}

	// windows that appear in cartridge specific menus
	if err := addWindow(newWinDPCregisters, false, windowMenuCart); err != nil {
		return nil, err
	}
	if err := addWindow(newWinDPCplusRegisters, false, windowMenuCart); err != nil {
		return nil, err
	}
	if err := addWindow(newWinSuperchargerRegisters, false, windowMenuCart); err != nil {
		return nil, err
	}
	if err := addWindow(newWinCartRAM, false, windowMenuCart); err != nil {
		return nil, err
	}
	if err := addWindow(newWinCartStatic, false, windowMenuCart); err != nil {
		return nil, err
	}
	if err := addWindow(newWinCartTape, false, windowMenuCart); err != nil {
		return nil, err
	}

	// plusrom windows
	if err := addWindow(newWinPlusROMNetwork, false, windowMenuOther); err != nil {
		return nil, err
	}
	if err := addWindow(newWinPlusROMPrefs, false, windowMenuOther); err != nil {
		return nil, err
	}

	// savekey windows
	if err := addWindow(newWinSaveKeyI2C, false, windowMenuOther); err != nil {
		return nil, err
	}
	if err := addWindow(newWinSaveKeyEEPROM, false, windowMenuOther); err != nil {
		return nil, err
	}

	// associate cartridge types with cartridge specific menus. using cartridge
	// ID as the key in the windowMenu map
	//
	// note that the name of the window menu's we use here must match the ID
	// used by the cartridge mapper.
	wm.windowMenu["DPC"] = append(wm.windowMenu["DPC"], winDPCregistersTitle)
	wm.windowMenu["DPC+"] = append(wm.windowMenu["DPC+"], winDPCplusRegistersTitle)
	wm.windowMenu["AR"] = append(wm.windowMenu["AR"], winSuperchargerRegistersTitle)

	// cartridges with RAM and static areas will be added automatically

	// get references to specific window types that need to be referenced
	// elsewhere in the system
	wm.dbgScr = wm.windows[winDbgScrTitle].(*winDbgScr)
	wm.term = wm.windows[winTermTitle].(*winTerm)

	// create play window. this is a very special window that never appears
	// directly in an any menu
	playWin := newWinPlayScr(img)
	wm.playScr = playWin.(*winPlayScr)

	// sort vcs menu entries
	sort.Strings(wm.windowMenu[windowMenuVCS])

	return wm, nil
}

func (wm *windowManager) init() {
	if wm.hasInitialised {
		return
	}

	for w := range wm.windows {
		wm.windows[w].init()
	}

	wm.hasInitialised = true
}

func (wm *windowManager) destroy() {
	for w := range wm.windows {
		wm.windows[w].destroy()
	}
}

func (wm *windowManager) draw() {
	if wm.img.lz.Dbg != nil {
		// there's no good place to call the init() function except during a
		// call to draw. the init() function itself handles
		wm.init()

		wm.drawMenu()
		for w := range wm.windows {
			wm.windows[w].draw()
		}
	}

	wm.playScr.draw()
}

func (wm *windowManager) drawMenu() {
	if !imgui.BeginMainMenuBar() {
		return
	}

	// see commentary for screenPos in windowManager declaration
	wm.screenPos = imgui.WindowPos()

	if imgui.BeginMenu(windowMenuDebugger) {
		for _, id := range wm.windowMenu[windowMenuDebugger] {
			wm.drawMenuWindowEntry(wm.windows[id], id)
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
		for _, id := range wm.windowMenu[windowMenuVCS] {
			wm.drawMenuWindowEntry(wm.windows[id], id)
		}

		imgui.EndMenu()
	}

	// add cartridge specific menu if cartridge has a RAM bus or a debug bus.
	// note that debug bus windows need to have been added to the window menu
	// for the specific cartridge ID. see newWindowManager() function above
	cartSpecificMenu := wm.img.lz.Cart.HasRAMbus ||
		wm.img.lz.Cart.HasStaticBus ||
		wm.img.lz.Cart.HasRegistersBus

	if _, ok := wm.windowMenu[wm.img.lz.Cart.ID]; ok {
		cartSpecificMenu = true
	}

	if cartSpecificMenu {
		if imgui.BeginMenu(fmt.Sprintf("Cart [%s]", wm.img.lz.Cart.ID)) {
			for _, id := range wm.windowMenu[wm.img.lz.Cart.ID] {
				wm.drawMenuWindowEntry(wm.windows[id], id)
			}

			if wm.img.lz.Cart.HasTapeBus {
				wm.drawMenuWindowEntry(wm.windows[winCartTapeTitle], winCartTapeTitle)
			}

			if wm.img.lz.Cart.HasRAMbus {
				wm.drawMenuWindowEntry(wm.windows[winCartRAMTitle], winCartRAMTitle)
			}

			if wm.img.lz.Cart.HasStaticBus {
				wm.drawMenuWindowEntry(wm.windows[winCartStaticTitle], winCartStaticTitle)
			}

			imgui.EndMenu()
		}
	}

	// plusrom specific menus
	if wm.img.lz.Cart.IsPlusROM {
		if imgui.BeginMenu("PlusROM") {
			wm.drawMenuWindowEntry(wm.windows[winPlusROMNetworkTitle], menuPlusROMNetworkTitle)
			wm.drawMenuWindowEntry(wm.windows[winPlusROMPrefsTitle], menuPlusROMPrefsTitle)
			imgui.EndMenu()
		}
	}

	// add savekey specific menu
	if wm.img.lz.SaveKey.SaveKeyActive {
		if imgui.BeginMenu("SaveKey") {
			wm.drawMenuWindowEntry(wm.windows[winSaveKeyI2CTitle], menuSaveKeyI2CTitle)
			wm.drawMenuWindowEntry(wm.windows[winSaveKeyEEPROMTitle], menuSaveKeyEEPROMTitle)
			imgui.EndMenu()
		}
	}

	// filename in titlebar
	imgui.SameLineV(imgui.WindowWidth()-imguiGetFrameDim(wm.img.lz.Cart.Filename).X-20.0, 0.0)
	imgui.Text(wm.img.lz.Cart.Filename)

	imgui.EndMainMenuBar()
}

func (wm *windowManager) drawMenuWindowEntry(w managedWindow, id string) {
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
