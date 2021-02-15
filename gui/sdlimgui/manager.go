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
	"github.com/inkyblackness/imgui-go/v3"
)

// the window type represents all the windows used in the sdlimgui.
type window interface {
	// initialisation function. by the first call to manager.draw()
	init()

	// id should return a unique identifier for the window. note that the
	// window title and any menu entry do not have to have the same value as
	// the id() but it can.
	id() string

	draw()
	isOpen() bool
	setOpen(bool)
}

// manager handles windows and menus in the system.
type manager struct {
	img *SdlImgui

	// has the window manager gone through the initialisation process
	hasInitialised bool

	// the collection of managed windows in the system, indexed by window title
	windows map[string]window

	// windows can be open and closed through the menu bar
	menu map[menuGroup][]menuEntry

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

	// some windows need to be referenced beyond the capabilities of the window
	// interface.
	//
	// if required, other windows can accessed by:
	//		window[title].(*windowType)
	//
	dbgScr   *winDbgScr
	crtPrefs *winCRTPrefs

	// the playscreen does not appear in the window list and can only be
	// referred to via the playScr field.
	playScr *winPlayScr
}

// windowDef specifies a window creator, the menu it appears in whether it
// should open on start.
type windowDef struct {
	// the window creation function
	create func(*SdlImgui) (window, error)

	// whether the window should be opened in an open or closed state
	open bool

	// menu entry the window will be associated with. the label and windowID
	// fields can be left blank. if it is the window.id() value will be used.
	//
	// the restrictBus and restrictMapper fields are optional.
	menu menuEntry
}

// list of windows to add to the window manager.
var windowDefs = [...]windowDef{
	// windows called from "debugger" menu
	{create: newFileSelector, menu: menuEntry{group: menuDebugger}},
	{create: newWinPrefs, menu: menuEntry{group: menuDebugger}},
	{create: newWinCRTPrefs, menu: menuEntry{group: menuDebugger}},
	{create: newWinRevisions, menu: menuEntry{group: menuDebugger}},
	{create: newWinTerm, menu: menuEntry{group: menuDebugger}},
	{create: newWinLog, menu: menuEntry{group: menuDebugger}},

	// windows that appear in the "vcs" menu
	{create: newWinAudio, menu: menuEntry{group: menuVCS}, open: true},
	{create: newWinChipRegisters, menu: menuEntry{group: menuVCS}},
	{create: newWinCollisions, menu: menuEntry{group: menuVCS}},
	{create: newWinControl, menu: menuEntry{group: menuVCS}, open: true},
	{create: newWinControllers, menu: menuEntry{group: menuVCS}},
	{create: newWinCPU, menu: menuEntry{group: menuVCS}, open: true},
	{create: newWinDisasm, menu: menuEntry{group: menuVCS}, open: true},
	{create: newWinDbgScr, menu: menuEntry{group: menuVCS}, open: true},
	{create: newWinRAM, menu: menuEntry{group: menuVCS}, open: true},
	{create: newWinTIA, menu: menuEntry{group: menuVCS}, open: true},
	{create: newWinTimer, menu: menuEntry{group: menuVCS}, open: true},

	// windows that appear in cartridge specific menu
	{create: newWinDPCregisters, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRegister, restrictMapper: []string{"DPC"}}},
	{create: newWinDPCplusRegisters, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRegister, restrictMapper: []string{"DPC+"}}},
	{create: newWinCDFRegisters, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRegister, restrictMapper: []string{"CDF"}}},
	{create: newWinSuperchargerRegisters, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRegister, restrictMapper: []string{"AR"}}},
	{create: newWinCartTape, menu: menuEntry{group: menuCart, restrictBus: menuRestrictTape}},
	{create: newWinCartRAM, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRAM}},
	{create: newWinCartStatic, menu: menuEntry{group: menuCart, restrictBus: menuRestrictStatic}},

	// coprocessor windows
	{create: newWinCoProcLastExecution, menu: menuEntry{group: menuCoProc, restrictBus: menuRestrictCoProc}},

	// plusrom windows
	{create: newWinPlusROMNetwork, menu: menuEntry{group: menuPlusROM, label: winPlusROMNetworkMenu}},
	{create: newWinPlusROMPrefs, menu: menuEntry{group: menuPlusROM, label: winPlusROMPrefsMenu}},

	// savekey windows
	{create: newWinSaveKeyI2C, menu: menuEntry{group: menuSaveKey, label: winSaveKeyI2CMenu}},
	{create: newWinSaveKeyEEPROM, menu: menuEntry{group: menuSaveKey, label: winSaveKeyEEPROMMenu}},
}

// list of windows that can be opened in playmode in addition to the debugger.
var playmodeWindows = [...]string{
	winCRTPrefsID,
	winTIARevisionsID,
}

func newManager(img *SdlImgui) (*manager, error) {
	wm := &manager{
		img:     img,
		windows: make(map[string]window),
		menu:    make(map[menuGroup][]menuEntry),
	}

	// create all window instances and add to specified menu
	for _, def := range windowDefs {
		w, err := def.create(img)
		if err != nil {
			return nil, err
		}
		wm.windows[w.id()] = w

		// open window if requested
		w.setOpen(def.open)

		// if menu label has not been specified use the window definition
		if def.menu.label == "" {
			def.menu.label = w.id()
		}

		// window name
		if def.menu.windowID == "" {
			def.menu.windowID = w.id()
		}

		// add menu entry
		wm.menu[def.menu.group] = append(wm.menu[def.menu.group], def.menu)
	}

	// get references to specific windows that need to be referenced elsewhere in the system
	wm.dbgScr = wm.windows[winDbgScrID].(*winDbgScr)
	wm.crtPrefs = wm.windows[winCRTPrefsID].(*winCRTPrefs)

	// create play window. this is a special window that does not appear in the window list
	wm.playScr = newWinPlayScr(img).(*winPlayScr)

	return wm, nil
}

func (wm *manager) draw() {
	// there's no good place to call the managedWindow.init() function except
	// here when we know everything else has been initialised
	if !wm.hasInitialised {
		for w := range wm.windows {
			wm.windows[w].init()
		}
		wm.hasInitialised = true
	}

	// playmode draws the screen and other windows that have been listed
	// as being safe to draw in playmode
	if wm.img.isPlaymode() {
		wm.playScr.draw()

		for _, s := range playmodeWindows {
			wm.windows[s].draw()
		}

		return
	}

	// no debugger is ready yet so return immediately
	if wm.img.lz.Dbg == nil {
		return
	}

	// draw menu
	wm.drawMenu()

	// draw windows
	for w := range wm.windows {
		wm.windows[w].draw()
	}
}
