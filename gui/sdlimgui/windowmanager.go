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
	"sort"

	"github.com/inkyblackness/imgui-go/v2"
)

// windowManagement can be embedded into a real window struct for
// basic window management functionality. it partially implements the
// managedWindow interface
type windowManagement struct {
	open bool
}

func (wm *windowManagement) isOpen() bool {
	return wm.open
}

func (wm *windowManagement) setOpen(open bool) {
	wm.open = open
}

// managedWindow conceptualises the functions required by a window such that
// it can be managed by the windowManager
type managedWindow interface {
	// init can be used to set up values that cannot be done during creation
	init()

	id() string
	destroy()
	draw()
	isOpen() bool
	setOpen(bool)
}

// windowManager is the nexus for all windows (including the main menu) in the
// imgui application
type windowManager struct {
	img *SdlImgui

	hasInitialised bool

	windows    map[string]managedWindow
	windowList []string

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

	// term and screen need to be accessfrom other areas of the package so we
	// maintain pointers to them in addition to there windows entries
	term *winTerm
	scr  *winScreen
}

func newWindowManager(img *SdlImgui) (*windowManager, error) {
	wm := &windowManager{
		img:        img,
		windows:    make(map[string]managedWindow),
		windowList: make([]string, 0),
	}

	addWindow := func(create func(img *SdlImgui) (managedWindow, error)) error {
		w, err := create(img)
		if err != nil {
			return err
		}

		wm.windows[w.id()] = w
		wm.windowList = append(wm.windowList, w.id())
		sort.Strings(wm.windowList)

		w.setOpen(true)

		return nil
	}

	if err := addWindow(newWinControl); err != nil {
		return nil, err
	}
	if err := addWindow(newWinCPU); err != nil {
		return nil, err
	}
	if err := addWindow(newWinRAM); err != nil {
		return nil, err
	}
	if err := addWindow(newWinTIA); err != nil {
		return nil, err
	}
	if err := addWindow(newWinTimer); err != nil {
		return nil, err
	}
	if err := addWindow(newWinDisasm); err != nil {
		return nil, err
	}
	if err := addWindow(newWinAudio); err != nil {
		return nil, err
	}
	if err := addWindow(newWinScreen); err != nil {
		return nil, err
	}
	if err := addWindow(newWinTerm); err != nil {
		return nil, err
	}

	wm.scr = wm.windows[winScreenTitle].(*winScreen)
	wm.term = wm.windows[winTermTitle].(*winTerm)

	return wm, nil
}

// init is called from drawWindows()
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

func (wm *windowManager) drawWindows() {
	if wm.img.lazy.VCS != nil && wm.img.lazy.Dsm != nil {
		wm.init()
		wm.drawMainMenu()
		for w := range wm.windows {
			wm.windows[w].draw()
		}
	}
}

func (wm *windowManager) drawMainMenu() {
	if imgui.BeginMainMenuBar() == false {
		return
	}

	// see commentary for screenPos in windowManager declaration
	wm.screenPos = imgui.WindowPos()

	if imgui.BeginMenu("Project") {
		if imgui.Selectable("Quit") {
			wm.img.term.pushCommand("QUIT")
		}
		imgui.EndMenu()
	}

	if imgui.BeginMenu("Windows") {
		for i := range wm.windowList {
			id := wm.windowList[i]
			if imgui.Selectable(id) {
				wm.windows[id].setOpen(true)
			}
		}

		imgui.EndMenu()
	}

	imgui.EndMainMenuBar()
}
