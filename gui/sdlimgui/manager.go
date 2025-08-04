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
	"strings"

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/imgui-go/v5"
)

// manager handles windows and menus in the system.
type manager struct {
	img *SdlImgui

	// has the window manager gone through the initialisation process
	hasInitialised bool

	// list of all windows
	windows map[string]window

	// playmode windows
	playmodeWindows map[string]playmodeWindow

	// debugger windows
	debuggerWindows map[string]debuggerWindow

	// list of windows that have been drawn most recently
	drawn map[string]bool

	// draw windows in order of size (smallest at the front) on the next draw()
	//
	// using int because we sometimes need to hold the arrangeBySize "signal"
	// in order for it to take effect. in most situations a value of 1 will be
	// sufficient for the arrangement to take place
	arrangeBySize int

	// windows can be open and closed through the menu bar
	menu map[menuGroup][]menuEntry

	// search windows
	searchActive bool
	searchString string

	// map of hotkeys. assigned as a result of window searching
	hotkeys map[rune]debuggerWindow

	// the window that is refocused and bought to the front when "captured running" is ended
	refocusWindow debuggerWindow

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

	// is true if mouse over any of the playmode windows
	playmodeCaptureInhibit bool

	// for convenience the dbgScr window gets it's own field
	//
	// if required, other windows can accessed by:
	//		window[title].(*windowType)
	dbgScr *winDbgScr
}

func newManager(img *SdlImgui) (*manager, error) {
	wm := &manager{
		img:             img,
		windows:         make(map[string]window),
		playmodeWindows: make(map[string]playmodeWindow),
		debuggerWindows: make(map[string]debuggerWindow),
		drawn:           make(map[string]bool),
		menu:            make(map[menuGroup][]menuEntry),
		hotkeys:         make(map[rune]debuggerWindow),
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

	return wm, nil
}

func (wm *manager) destroy() {
	for _, w := range wm.windows {
		if c, ok := w.(windowDestroy); ok {
			c.destroy()
		}
	}
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

	switch wm.img.mode.Load().(govern.Mode) {
	case govern.ModePlay:
		// reset playmodeHover flag by default. it's only ever true if a window is open (and that
		// window is being hovered over)
		wm.playmodeCaptureInhibit = false

		// playmode draws the screen and other windows that have been listed
		// as being safe to draw in playmode
		for _, w := range wm.playmodeWindows {
			if w.playmodeDraw() {
				wm.playmodeCaptureInhibit = wm.playmodeCaptureInhibit || w.playmodeIsHovered()
			}
		}

		// inhibit playmode capture if any popup is open
		wm.playmodeCaptureInhibit = wm.playmodeCaptureInhibit || imgui.IsPopupOpenV("", imgui.PopupFlagsAnyPopup)

	case govern.ModeDebugger:
		// see commentary for screenPos in windowManager declaration
		wm.screenPos = imgui.WindowPos()

		// no debugger is not ready yet so return immediately
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
				gi := l[i].debuggerGeometry().size
				gj := l[j].debuggerGeometry().size
				return gi.X*gi.Y > gj.X*gj.X
			})

			// drawing every window with window focus set will cause an ugly
			// colour flash in the title bars. push the inactive color to the
			// active color
			imgui.PushStyleColor(imgui.StyleColorTitleBgActive, imgui.CurrentStyle().Color(imgui.StyleColorTitleBg))

			// draw in order of size
			for _, w := range l {
				imgui.SetNextWindowFocus()
				w.debuggerDraw()
			}

			// undo earlier style push
			imgui.PopStyleColor()

		} else {
			var searchCandidates []string

			for _, w := range wm.debuggerWindows {
				geom := w.debuggerGeometry()

				// raise window to front of display
				if w.debuggerGeometry().raiseOnNextDraw {
					imgui.SetNextWindowFocus()
					geom.raiseOnNextDraw = false
				}

				// draw window
				wm.drawn[w.id()] = w.debuggerDraw()

				// set window to refocus on
				if geom.focused && !geom.noFocusTracking {
					wm.refocusWindow = w
				}

				// add to list of search candidates
				if wm.searchActive && wm.drawn[w.id()] {
					searchCandidates = append(searchCandidates, w.id())
				}
			}

			if wm.searchActive {
				var match debuggerWindow

				for _, o := range searchCandidates {
					if strings.Contains(strings.ToLower(o), wm.searchString) {
						if match != nil {
							match = nil
							break // for loop
						}
						match = wm.debuggerWindows[o]
					}
				}

				// no match so use the hotkeys array to select a window
				if match == nil && len(wm.searchString) > 0 {
					match = wm.hotkeys[rune(wm.searchString[0])]
					if match != nil {
						// window has been closed since the preference was set
						if !wm.drawn[match.id()] {
							match = nil
						}
					}
				}

				// raise matched window on next draw
				if match != nil {
					geom := match.debuggerGeometry()
					geom.raiseOnNextDraw = true

					// update hotkeys information
					wm.hotkeys[rune(wm.searchString[0])] = match
				}
			} else {
				wm.searchString = ""
			}
		}
	}
}

// return true if the mouse pointer is hovering over any playmode window.
// returns false if the mouse is over the main playmode TV screen
func (wm *manager) hoverAnyWindowPlaymode() bool {
	for _, w := range wm.playmodeWindows {
		if w.playmodeIsHovered() {
			return true
		}
	}
	return false
}
