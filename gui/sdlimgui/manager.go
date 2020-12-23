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

	"github.com/inkyblackness/imgui-go/v2"
)

// window represents all the window types used in the sdlimgui.
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
	playScr  *winPlayScr
}

func newManager(img *SdlImgui) (*manager, error) {
	wm := &manager{
		img:     img,
		windows: make(map[string]window),
		menu:    make(map[string][]string),
	}

	// creation function for all managed windows
	addWindow := func(create func(img *SdlImgui) (window, error), open bool, group string) error {
		w, err := create(img)
		if err != nil {
			return err
		}

		wm.windows[w.id()] = w
		wm.menu[group] = append(wm.menu[group], w.id())
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
	if err := addWindow(newWinCRTPrefs, false, windowMenuDebugger); err != nil {
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
	if err := addWindow(newWinCDFRegisters, false, windowMenuCart); err != nil {
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
	wm.menu["DPC"] = append(wm.menu["DPC"], winDPCregistersTitle)
	wm.menu["DPC+"] = append(wm.menu["DPC+"], winDPCplusRegistersTitle)
	wm.menu["AR"] = append(wm.menu["AR"], winSuperchargerRegistersTitle)
	wm.menu["CDF"] = append(wm.menu["CDF"], winCDFRegistersTitle)

	// cartridges with RAM and static areas will be added automatically

	// get references to specific window types that need to be referenced
	// elsewhere in the system
	wm.dbgScr = wm.windows[winDbgScrTitle].(*winDbgScr)
	wm.crtPrefs = wm.windows[winCRTPrefsTitle].(*winCRTPrefs)

	// create play window. this is a very special window that never appears
	// directly in an any menu
	wm.playScr = newWinPlayScr(img).(*winPlayScr)

	// sort vcs menu entries. leave other menus alone
	sort.Strings(wm.menu[windowMenuVCS])

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

		// we wont' be initialising again
		wm.hasInitialised = true
	}

	// playmode is very simple
	if wm.img.isPlaymode() {
		wm.playScr.draw()
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
