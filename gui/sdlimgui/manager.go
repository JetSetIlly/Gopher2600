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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/debugger/govern"
)

// manager handles windows and menus in the system.
type manager struct {
	img *SdlImgui

	// has the window manager gone through the initialisation process
	hasInitialised bool

	windows map[string]window

	// playmode windows
	playmodeWindows map[string]playmodeWindow

	// debugger debuggerWindows
	debuggerWindows map[string]debuggerWindow

	// draw windows in order of size (smallest at the front) on the next draw()
	//
	// using int because we sometimes need to hold the arrangeBySize "signal"
	// in order for it to take effect. in most situations a value of 1 will be
	// sufficient for the arrangement to take place
	arrangeBySize int

	// windows can be open and closed through the menu bar
	menu map[menuGroup][]menuEntry

	// some windows need to be referenced beyond the capabilities of the window
	// interface.
	//
	// if required, other windows can accessed by:
	//		window[title].(*windowType)
	//
	// the following fields are provided for convenience.
	dbgScr   *winDbgScr
	timeline *winTimeline

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

// windowDef specifies a window creator, the menu it appears in whether it
// should open on start.
type windowDef struct {
	// the window creation function
	create func(*SdlImgui) (window, error)

	// menu entry the window will be associated with. the label and windowID
	// fields can be left blank. if it is the window.id() value will be used.
	//
	// the restrictBus and restrictMapper fields are optional.
	menu menuEntry

	// whether the window should be opened in an defaultOpen or closed state. this is
	// the default state and will be overridden if loadManagerState() is called
	// and a previously saved state can be found
	defaultOpen bool
}

// list of windows to add to the window manager.
var windowDefs = [...]windowDef{
	// windows called from "debugger" menu
	{create: newFileSelector, menu: menuEntry{group: menuDebugger}},
	{create: newWinPrefs, menu: menuEntry{group: menuDebugger}},
	{create: newWinTerm, menu: menuEntry{group: menuDebugger}},
	{create: newWinLog, menu: menuEntry{group: menuDebugger}},

	// windows that appear in the "vcs" menu
	{create: newWinCollisions, menu: menuEntry{group: menuVCS}},
	{create: newWinControl, menu: menuEntry{group: menuVCS}, defaultOpen: true},
	{create: newWinCPU, menu: menuEntry{group: menuVCS}, defaultOpen: true},
	{create: newWinDisasm, menu: menuEntry{group: menuVCS}, defaultOpen: true},
	{create: newWinDbgScr, menu: menuEntry{group: menuVCS}, defaultOpen: true},
	{create: newWinRAM, menu: menuEntry{group: menuVCS}, defaultOpen: true},
	{create: newWinPeripherals, menu: menuEntry{group: menuVCS}},
	{create: newWinPorts, menu: menuEntry{group: menuVCS}},
	{create: newWinTIA, menu: menuEntry{group: menuVCS}, defaultOpen: true},
	{create: newWinTimer, menu: menuEntry{group: menuVCS}, defaultOpen: true},

	// windows that appear in the "tools" menu
	{create: newWinOscilloscope, menu: menuEntry{group: menuTools}, defaultOpen: true},
	{create: newWinTracker, menu: menuEntry{group: menuTools}, defaultOpen: false},
	{create: newWinTimeline, menu: menuEntry{group: menuTools}, defaultOpen: true},
	{create: newWin6507Pinout, menu: menuEntry{group: menuTools}, defaultOpen: false},
	{create: newWinPaint, menu: menuEntry{group: menuTools}, defaultOpen: false},

	// windows that appear in cartridge specific menu
	{create: newWinDPCregisters, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRegister, restrictMapper: []string{"DPC"}}},
	{create: newWinDPCplusRegisters, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRegister, restrictMapper: []string{"DPC+"}}},
	{create: newWinCDFRegisters, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRegister, restrictMapper: []string{"CDF", "CDFJ", "CDF0", "CDF1", "CDFJ+"}}},
	{create: newWinCDFStreams, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRegister, restrictMapper: []string{"CDF", "CDFJ", "CDF0", "CDF1", "CDFJ+"}}},
	{create: newWinSuperchargerRegisters, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRegister, restrictMapper: []string{"AR"}}},
	{create: newWinCartTape, menu: menuEntry{group: menuCart, restrictBus: menuRestrictTape}},
	{create: newWinCartRAM, menu: menuEntry{group: menuCart, restrictBus: menuRestrictRAM}},
	{create: newWinCartStatic, menu: menuEntry{group: menuCart, restrictBus: menuRestrictStatic}},

	// coprocessor windows
	{create: newWinCoProcDisasm, menu: menuEntry{group: menuCoProc, restrictBus: menuRestrictCoProc, label: winCoProcDisasmMenu}},
	{create: newWinCoProcIllegalAccess, menu: menuEntry{group: menuCoProc, restrictBus: menuRestrictCoProc, label: winCoProcIllegalAccessMenu}},
	{create: newWinCoProcPerformance, menu: menuEntry{group: menuCoProc, restrictBus: menuRestrictCoProc, label: winCoProcPerformanceMenu}},
	{create: newWinCoProcGlobals, menu: menuEntry{group: menuCoProc, restrictBus: menuRestrictCoProc, label: winCoProcGlobalsMenu}},
	{create: newWinCoProcLocals, menu: menuEntry{group: menuCoProc, restrictBus: menuRestrictCoProc, label: winCoProcLocalsMenu}},
	{create: newWinCoProcSource, menu: menuEntry{group: menuCoProc, restrictBus: menuRestrictCoProc, label: winCoProcSourceMenu}},

	// plusrom windows
	{create: newWinPlusROMNetwork, menu: menuEntry{group: menuPlusROM, label: winPlusROMNetworkMenu}},
	{create: newWinPlusROMNick, menu: menuEntry{group: menuPlusROM, label: winPlusROMNickMenu}},

	// savekey windows
	{create: newWinSaveKeyI2C, menu: menuEntry{group: menuSaveKey, label: winSaveKeyI2CMenu}},
	{create: newWinSaveKeyEEPROM, menu: menuEntry{group: menuSaveKey, label: winSaveKeyEEPROMMenu}},

	// windows that do not have a menu entry (windows for playmode only)
	{create: newWinComparison, menu: menuEntry{group: menuNone}, defaultOpen: false},
	{create: newWinBot, menu: menuEntry{group: menuNone}, defaultOpen: false},
}

func newManager(img *SdlImgui) (*manager, error) {
	wm := &manager{
		img:             img,
		windows:         make(map[string]window),
		playmodeWindows: make(map[string]playmodeWindow),
		debuggerWindows: make(map[string]debuggerWindow),
		menu:            make(map[menuGroup][]menuEntry),
	}

	// create all window instances and add to specified menu
	for _, def := range windowDefs {
		w, err := def.create(img)
		if err != nil {
			return nil, err
		}

		wm.windows[w.id()] = w

		// all windows that implement the playmodeWindow interface will be
		// added to the list of playmode windows
		if pw, ok := w.(playmodeWindow); ok {
			wm.playmodeWindows[pw.id()] = pw
		}

		// all windows that implement the debuggerWindow interface will be
		// added to the list of debugger windows
		if dw, ok := w.(debuggerWindow); ok {
			wm.debuggerWindows[w.id()] = dw

			// default window state
			dw.debuggerSetOpen(def.defaultOpen)

			// if menu label has not been specified use the window definition
			if def.menu.label == "" {
				def.menu.label = dw.id()
			}

			// window name
			if def.menu.windowID == "" {
				def.menu.windowID = dw.id()
			}

			// add menu entry
			wm.menu[def.menu.group] = append(wm.menu[def.menu.group], def.menu)
		}
	}

	// get references to specific windows that need to be referenced elsewhere in the system
	wm.dbgScr = wm.debuggerWindows[winDbgScrID].(*winDbgScr)
	wm.timeline = wm.debuggerWindows[winTimelineID].(*winTimeline)

	return wm, nil
}

func (wm *manager) draw() {
	// there's no good place to call the managedWindow.init() function except
	// here when we know everything else has been initialised
	if !wm.hasInitialised {
		for w := range wm.debuggerWindows {
			wm.debuggerWindows[w].init()
		}
		wm.hasInitialised = true
	}

	switch wm.img.mode {
	case govern.ModePlay:
		// playmode draws the screen and other windows that have been listed
		// as being safe to draw in playmode
		for _, w := range wm.playmodeWindows {
			w.playmodeDraw()
		}
	case govern.ModeDebugger:
		// see commentary for screenPos in windowManager declaration
		wm.screenPos = imgui.WindowPos()

		// no debugger is ready yet so return immediately
		if wm.img.dbg == nil {
			return
		}

		// draw menu
		wm.drawMenu()

		// draw windows
		if wm.arrangeBySize > 0 {
			wm.arrangeBySize--

			// sort windows in order of size smallest at the front
			l := make([]debuggerWindow, 0, len(wm.debuggerWindows))
			for _, w := range wm.debuggerWindows {
				l = append(l, w)
			}

			sort.Slice(l, func(i int, j int) bool {
				gi := l[i].debuggerGeometry().windowSize
				gj := l[j].debuggerGeometry().windowSize
				return gi.X*gi.Y > gj.X*gj.X
			})

			// drawing every window with window focus set will cause an ugly
			// colour flash in the title bars. push the inactive color to the
			// active color
			sty := imgui.CurrentStyle()
			imgui.PushStyleColor(imgui.StyleColorTitleBgActive, sty.Color(imgui.StyleColorTitleBg))

			// draw in order of size
			for _, w := range l {
				imgui.SetNextWindowFocus()
				w.debuggerDraw()
			}

			// undo early style push
			imgui.PopStyleColor()

		} else {
			for _, w := range wm.debuggerWindows {
				w.debuggerDraw()
			}
		}
	}
}
