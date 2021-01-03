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
	"sort"

	"github.com/inkyblackness/imgui-go/v3"
)

// the window type represents all the windows used in the sdlimgui.
type window interface {
	init()
	id() string
	destroy()
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

	// windows can be open and closed through the menu bar. they are grouped
	// according to type using the menu constants defined below.
	menu map[string][]string

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
// should open on start
type windowDef struct {
	create func(*SdlImgui) (window, error)
	menu   string
	open   bool
}

// list of windows to add to the window manager
var windowDefs = [...]windowDef{
	// windows called from "debugger" menu
	{create: newFileSelector, menu: menuDebugger},
	{create: newWinPrefs, menu: menuDebugger},
	{create: newWinCRTPrefs, menu: menuDebugger},
	{create: newWinTerm, menu: menuDebugger},
	{create: newWinLog, menu: menuDebugger},

	// windows that appear in the "vcs" menu
	{create: newWinControl, menu: menuVCS, open: true},
	{create: newWinCPU, menu: menuVCS, open: true},
	{create: newWinRAM, menu: menuVCS, open: true},
	{create: newWinTIA, menu: menuVCS, open: true},
	{create: newWinTimer, menu: menuVCS, open: true},
	{create: newWinDisasm, menu: menuVCS, open: true},
	{create: newWinAudio, menu: menuVCS, open: true},
	{create: newWinDbgScr, menu: menuVCS, open: true},
	{create: newWinControllers, menu: menuVCS},
	{create: newWinCollisions, menu: menuVCS},
	{create: newWinChipRegisters, menu: menuVCS},

	// windows that appear in cartridge specific menus
	{create: newWinDPCregisters, menu: "DPC"},
	{create: newWinDPCplusRegisters, menu: "DPC+"},
	{create: newWinCDFRegisters, menu: "CDF"},
	{create: newWinSuperchargerRegisters, menu: "AR"},
	{create: newWinCartTape, menu: "AR"},
	{create: newWinCartRAM, menu: menuCart},
	{create: newWinCartStatic, menu: menuCart},

	// cartridges with RAM and static areas will be added automatically

	// plusrom windows
	{create: newWinPlusROMNetwork, menu: menuPlusROM},
	{create: newWinPlusROMPrefs, menu: menuPlusROM},

	// savekey windows
	{create: newWinSaveKeyI2C, menu: menuSaveKey},
	{create: newWinSaveKeyEEPROM, menu: menuSaveKey},
}

// list of windows that can be opened in playmode in addition to the debugger
var playmodeWindows = [...]string{
	winCRTPrefsTitle,
}

func newManager(img *SdlImgui) (*manager, error) {
	wm := &manager{
		img:     img,
		windows: make(map[string]window),
		menu:    make(map[string][]string),
	}

	// create all window instances and add to specified menu
	addWindow := func(def windowDef) error {
		w, err := def.create(img)
		if err != nil {
			return err
		}

		wm.windows[w.id()] = w
		wm.menu[def.menu] = append(wm.menu[def.menu], w.id())
		w.setOpen(def.open)

		return nil
	}

	for _, w := range windowDefs {
		if err := addWindow(w); err != nil {
			return nil, err
		}
	}

	// sort vcs menu entries. leave other menus alone
	sort.Strings(wm.menu[menuVCS])

	// get references to specific windows that need to be referenced elsewhere in the system
	wm.dbgScr = wm.windows[winDbgScrTitle].(*winDbgScr)
	wm.crtPrefs = wm.windows[winCRTPrefsTitle].(*winCRTPrefs)

	// create play window. this is a special window that does not appear in the
	// window list
	wm.playScr = newWinPlayScr(img).(*winPlayScr)

	return wm, nil
}

func (wm *manager) destroy() {
	for w := range wm.windows {
		wm.windows[w].destroy()
	}
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
