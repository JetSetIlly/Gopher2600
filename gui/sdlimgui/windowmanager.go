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

	windows map[string]managedWindow

	// term and screen need to be accessfrom other areas of the package so we
	// maintain pointers to them in addition to there windows entries
	term *term
	scr  *screen
}

func newWindowManager(img *SdlImgui) (*windowManager, error) {
	wm := &windowManager{
		img:     img,
		windows: make(map[string]managedWindow),
	}

	addWindow := func(create func(img *SdlImgui) (managedWindow, error)) error {
		w, err := create(img)
		if err != nil {
			return err
		}
		wm.windows[w.id()] = w
		w.setOpen(true)
		return nil
	}

	if err := addWindow(newMainMenu); err != nil {
		return nil, err
	}

	if err := addWindow(newControl); err != nil {
		return nil, err
	}
	if err := addWindow(newCPU); err != nil {
		return nil, err
	}
	if err := addWindow(newRAM); err != nil {
		return nil, err
	}
	if err := addWindow(newDelays); err != nil {
		return nil, err
	}
	if err := addWindow(newTIA); err != nil {
		return nil, err
	}
	if err := addWindow(newRIOT); err != nil {
		return nil, err
	}
	if err := addWindow(newDisasm); err != nil {
		return nil, err
	}
	if err := addWindow(newOscilloscope); err != nil {
		return nil, err
	}
	if err := addWindow(newTvScreen); err != nil {
		return nil, err
	}
	if err := addWindow(newTerm); err != nil {
		return nil, err
	}

	wm.scr = wm.windows[screenTitle].(*screen)
	wm.term = wm.windows[termTitle].(*term)

	return wm, nil
}

func (wm *windowManager) destroy() {
	for w := range wm.windows {
		wm.windows[w].destroy()
	}
}

func (wm *windowManager) drawWindows() {
	if wm.img.vcs != nil {
		for w := range wm.windows {
			wm.windows[w].draw()
		}
	}
}
