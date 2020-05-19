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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlimgui

import (
	"fmt"
	"sort"

	"github.com/inkyblackness/imgui-go/v2"
)

// managedWindow conceptualises the functions required by a window such that
// it can be managed by the windowManager
type managedWindow interface {
	init()
	id() string
	destroy()
	draw()
	isOpen() bool
	setOpen(bool)
}

// windowManager handles windows and menus in the system
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

// the window menus grouped by type. the types are:
const (
	windowMenuProject = "Project"
	windowMenuMain    = "Windows"
	windowMenuCart    = "Cartridge"

	// additional window menus are grouped by cartridge type
)

func newWindowManager(img *SdlImgui) (*windowManager, error) {
	wm := &windowManager{
		img:        img,
		windows:    make(map[string]managedWindow),
		windowMenu: make(map[string][]string, 0),
	}

	// creation function for all managed windows
	addWindow := func(create func(img *SdlImgui) (managedWindow, error), open bool, group string) error {
		w, err := create(img)
		if err != nil {
			return err
		}

		wm.windows[w.id()] = w
		wm.windowMenu[group] = append(wm.windowMenu[group], w.id())
		sort.Strings(wm.windowMenu[group])

		w.setOpen(open)

		return nil
	}

	// windows called from project menu
	if err := addWindow(newFileSelector, false, windowMenuProject); err != nil {
		return nil, err
	}
	if err := addWindow(newWinPrefs, false, windowMenuProject); err != nil {
		return nil, err
	}

	// windows that appear in the "windows" menu
	if err := addWindow(newWinControl, true, windowMenuMain); err != nil {
		return nil, err
	}
	if err := addWindow(newWinCPU, true, windowMenuMain); err != nil {
		return nil, err
	}
	if err := addWindow(newWinRAM, true, windowMenuMain); err != nil {
		return nil, err
	}
	if err := addWindow(newWinTIA, true, windowMenuMain); err != nil {
		return nil, err
	}
	if err := addWindow(newWinTimer, true, windowMenuMain); err != nil {
		return nil, err
	}
	if err := addWindow(newWinDisasm, true, windowMenuMain); err != nil {
		return nil, err
	}
	if err := addWindow(newWinAudio, true, windowMenuMain); err != nil {
		return nil, err
	}
	if err := addWindow(newWinDbgScr, true, windowMenuMain); err != nil {
		return nil, err
	}
	if err := addWindow(newWinTerm, false, windowMenuMain); err != nil {
		return nil, err
	}
	if err := addWindow(newWinControllers, false, windowMenuMain); err != nil {
		return nil, err
	}

	// associate cartridge types with cartridge specific menus. using cartridge
	// ID as the key in the windowMenu map
	wm.windowMenu["DPC"] = append(wm.windowMenu["DPC"], winStaticTitle)

	// get references to specific window types that need to be referenced
	// elsewhere in the system
	wm.dbgScr = wm.windows[winDbgScrTitle].(*winDbgScr)
	wm.term = wm.windows[winTermTitle].(*winTerm)

	// create play window. this is a very special window that never appears
	// directly in an any menu
	playWin, err := newWinPlayScr(img)
	if err != nil {
		return nil, err
	}
	wm.playScr = playWin.(*winPlayScr)

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
	if wm.img.lz.VCS != nil && wm.img.lz.Dsm != nil {
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
	if imgui.BeginMainMenuBar() == false {
		return
	}

	// see commentary for screenPos in windowManager declaration
	wm.screenPos = imgui.WindowPos()

	if imgui.BeginMenu("Project") {
		for _, id := range wm.windowMenu[windowMenuProject] {
			w := wm.windows[id]

			// decorate the menu entry with elipsis
			if imgui.Selectable(fmt.Sprintf("%s...", id)) {
				// windows in this menu will open on select
				w.setOpen(true)
			}
		}
		if imgui.Selectable("Quit") {
			wm.img.term.pushCommand("QUIT")
		}
		imgui.EndMenu()
	}

	// window menu
	if imgui.BeginMenu(windowMenuMain) {
		for _, id := range wm.windowMenu[windowMenuMain] {
			// add decorator indicating if window is currently open
			w := wm.windows[id]
			if w.isOpen() {
				// checkmark is unicode middle dot - code 00b7
				id = fmt.Sprintf("· %s", id)
			} else {
				id = fmt.Sprintf("  %s", id)
			}

			if imgui.Selectable(id) {
				// windows in this menu are toggleable
				if w.isOpen() {
					w.setOpen(false)
				} else {
					w.setOpen(true)
				}
			}
		}

		imgui.EndMenu()
	}

	// add cartridge specific menu
	if l, ok := wm.windowMenu[wm.img.lz.Cart.ID]; ok {
		if imgui.BeginMenu(wm.img.lz.Cart.ID) {
			for _, id := range l {
				w := wm.windows[id]

				// decorate the menu entry with an "window open" indicator
				if w.isOpen() {
					// checkmark is unicode middle dot - code 00b7
					id = fmt.Sprintf("· %s", id)
				} else {
					id = fmt.Sprintf("  %s", id)
				}

				if imgui.Selectable(id) {
					// windows in this menu are toggleable
					if w.isOpen() {
						w.setOpen(false)
					} else {
						w.setOpen(true)
					}
				}
			}

			imgui.EndMenu()
		}
	}

	imgui.EndMainMenuBar()
}
