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
	"strings"
	"time"
	"unicode"

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/userinput"
	"github.com/jetsetilly/imgui-go/v5"
	"github.com/veandco/go-sdl2/sdl"
)

func (img *SdlImgui) serviceKeyboard(ev *sdl.KeyboardEvent) {
	// notify phantom input system of a backspace. placed before the
	// no-repeat-key-presses test below
	if ev.Keysym.Scancode == sdl.SCANCODE_BACKSPACE && ev.Type == sdl.KEYDOWN {
		img.phantomInput = phantomInputBackSpace
	}

	// most keyboard events want to ignore repeat events. any that don't should
	// be handled before this condition
	if ev.Repeat == 1 {
		return
	}

	ctrl := ev.Keysym.Mod&sdl.KMOD_LCTRL == sdl.KMOD_LCTRL || ev.Keysym.Mod&sdl.KMOD_RCTRL == sdl.KMOD_RCTRL
	alt := ev.Keysym.Mod&sdl.KMOD_LALT == sdl.KMOD_LALT || ev.Keysym.Mod&sdl.KMOD_RALT == sdl.KMOD_RALT
	shift := ev.Keysym.Mod&sdl.KMOD_LSHIFT == sdl.KMOD_LSHIFT || ev.Keysym.Mod&sdl.KMOD_RSHIFT == sdl.KMOD_RSHIFT

	// enable window searching based on keyboard modifiers
	img.wm.searchActive = ctrl && shift

	handled := true

	if ev.Type == sdl.KEYUP {
		// special handling if window searching is enabled
		if img.wm.searchActive {
			switch ev.Keysym.Scancode {
			case sdl.SCANCODE_BACKSPACE:
				if len(img.wm.searchString) > 0 {
					img.wm.searchString = img.wm.searchString[:len(img.wm.searchString)-1]
				}
			case sdl.SCANCODE_SPACE:
				// dont allow leading space
				if len(img.wm.searchString) > 0 {
					img.wm.searchString = fmt.Sprintf("%s ", img.wm.searchString)
				}
			default:
				key := sdl.GetKeyFromScancode(ev.Keysym.Scancode)
				if unicode.IsPrint(rune(key)) {
					name := sdl.GetScancodeName(ev.Keysym.Scancode)
					img.wm.searchString = fmt.Sprintf("%s%s", img.wm.searchString, strings.ToLower(name))
				}
			}
			return
		}

		if img.isPlaymode() {
			switch ev.Keysym.Scancode {
			case sdl.SCANCODE_ESCAPE:
				// the escape key is useful to the keyportari so we forward it to the main emulation
				// unless the shift key is pressed
				if !shift && (img.cache.VCS.RIOT.Ports.LeftPlayer.ID() == plugging.PeriphKeyportari ||
					img.cache.VCS.RIOT.Ports.RightPlayer.ID() == plugging.PeriphKeyportari) {
					handled = false
				} else {
					if img.isCaptured() {
						img.setCapture(false)
					} else if img.wm.playmodeWindows[winSelectROMID].playmodeIsOpen() {
						img.wm.playmodeWindows[winSelectROMID].playmodeSetOpen(false)
					} else {
						img.quit()
					}
				}

			case sdl.SCANCODE_LEFT:
				if alt {
					img.screen.SetRotation(specification.LeftRotation)
				}
			case sdl.SCANCODE_RIGHT:
				if alt {
					img.screen.SetRotation(specification.RightRotation)
				}
			case sdl.SCANCODE_UP:
				if alt {
					img.screen.SetRotation(specification.NormalRotation)
				}
			case sdl.SCANCODE_DOWN:
				if alt {
					img.screen.SetRotation(specification.FlippedRotation)
				}

			default:
				handled = false
			}
		}

		if !img.modalActive() {
			switch ev.Keysym.Scancode {
			case sdl.SCANCODE_TAB:
				if !img.isPlaymode() && imgui.IsAnyItemActive() {
					// in debugger mode do not handle if an imgui widget is active
					// (see the sdl.KEYDOWN branch below for opposite condition and
					// explanation)
					handled = false
				} else {
					if ctrl {
						img.dbg.ReloadCartridge()
					} else {
						// only open ROM selector if window has been focused for a
						// while. see windowFocusedTime declaration for an explanation
						if time.Since(img.windowFocusedTime) > 500*time.Millisecond {
							img.wm.toggleOpen(winSelectROMID)
						}
					}
				}

			case sdl.SCANCODE_GRAVE:
				img.suppressTextInput = false
				if img.isPlaymode() {
					img.dbg.PushSetMode(govern.ModeDebugger)
				} else {
					img.dbg.PushSetMode(govern.ModePlay)
				}

			case sdl.SCANCODE_F7:
				if img.isPlaymode() {
					fps := img.prefs.fpsDetail.Get().(bool)
					img.prefs.fpsDetail.Set(!fps)
				}

			case sdl.SCANCODE_F8:
				w := img.wm.playmodeWindows[winBotID]
				w.playmodeSetOpen(!w.playmodeIsOpen())

			case sdl.SCANCODE_F9:
				img.wm.toggleOpen(winTrackerID)

			case sdl.SCANCODE_F10:
				img.wm.toggleOpen(winPrefsID)

			case sdl.SCANCODE_F11:
				img.prefs.fullScreen.Set(!img.prefs.fullScreen.Get().(bool))

			case sdl.SCANCODE_F12:
				if alt && !ctrl && !shift {
					img.screenshot(modeMovement, "")
				} else if ctrl && !shift && !alt {
					img.screenshot(modeTriple, "")
				} else if shift && !ctrl && !alt {
					img.screenshot(modeDouble, "")
				} else {
					img.screenshot(modeSingle, "")
				}

			case sdl.SCANCODE_F14:
				fallthrough
			case sdl.SCANCODE_SCROLLLOCK:
				img.setCapture(!img.isCaptured())

			case sdl.SCANCODE_F15:
				fallthrough
			case sdl.SCANCODE_PAUSE:
				if img.isPlaymode() {
					if img.dbg.State() == govern.Paused {
						img.dbg.PushSetPause(false)
					} else {
						img.dbg.PushSetPause(true)
					}
				} else {
					if img.dbg.State() == govern.Paused {
						img.term.pushCommand("RUN")
					} else {
						img.setCapturedRunning(false)
					}
				}

			case sdl.SCANCODE_A:
				if ctrl {
					img.wm.arrangeBySize = 1
				} else {
					handled = false
				}

			case sdl.SCANCODE_R:
				if ctrl {
					img.dbg.ReloadCartridge()
				} else {
					handled = false
				}

			case sdl.SCANCODE_M:
				if ctrl {
					if alt {
						img.dbg.PushMemoryProfile()
					} else {
						img.toggleAudioMute()
					}
				} else {
					handled = false
				}

			default:
				handled = false
			}
		}
	}

	if ev.Type == sdl.KEYDOWN {
		if !img.modalActive() {
			switch ev.Keysym.Scancode {
			case sdl.SCANCODE_TAB:
				if !imgui.IsAnyItemActive() {
					return
				}
			case sdl.SCANCODE_ESCAPE:
				// for debugger mode we test for the ESC key press on the down event
				// and not the up event. this is because imgui widgets react to the ESC
				// key on the down event and we only want to perform our special ESC
				// key handling if no widget is active
				//
				// if we perform out special handling on the up stroke then the active
				// widget will be unselected and then the special handling perfomed on
				// every ESC KEY press. we don't want that. we want the active widget
				// to be deselected and for the special handling to require a
				// completely separate key press
				if !img.isPlaymode() && !imgui.IsAnyItemActive() {
					img.setCapturedRunning(!img.isCapturedRunning())
				}
			case sdl.SCANCODE_GRAVE:
				// this is the key we use to switch playmode & debugger. it's okay to forward this
				// to the userinput except when keyportari is in use, when the grave can introduce
				// bogus input
				img.suppressTextInput = true
			default:
				handled = false
			}
		}
	}

	// early return if keypress has been handled
	if handled {
		return
	}

	// forward keypresses to userinput.Event channel
	if img.isCaptured() || (img.isPlaymode() && !imgui.IsAnyItemActive()) {
		switch ev.Type {
		case sdl.KEYDOWN:
			fallthrough
		case sdl.KEYUP:
			key := sdl.GetScancodeName(ev.Keysym.Scancode)

			// TODO: keyboard mapping sensitive. this is required for keyportari protocols

			// for some reason GetScancodeName() does not work for all symbols
			if key == "" {
				switch ev.Keysym.Scancode {
				case 100:
					key = "\\"
				}
			}

			select {
			case img.dbg.UserInput() <- userinput.EventKeyboard{
				Key:  key,
				Down: ev.Type == sdl.KEYDOWN,
				Mod:  getKeyMod(),
			}:
			default:
				logger.Log(logger.Allow, "sdlimgui", "dropped keyboard event")
			}
		}
	}

	// remaining keypresses forwarded to imgui io system
	k := sdl2KeyEventToImguiKey(ev.Keysym.Sym, ev.Keysym.Scancode)
	io := imgui.CurrentIO()
	switch ev.Type {
	case sdl.KEYDOWN:
		io.AddKeyEvent(k, true)
	case sdl.KEYUP:
		io.AddKeyEvent(k, false)
	}
	sdl2SetImguiModKey(io, ev.Keysym.Mod)
}

func getKeyMod() userinput.KeyMod {
	if sdl.GetModState()&sdl.KMOD_LALT == sdl.KMOD_LALT ||
		sdl.GetModState()&sdl.KMOD_RALT == sdl.KMOD_RALT {
		return userinput.KeyModAlt
	} else if sdl.GetModState()&sdl.KMOD_LSHIFT == sdl.KMOD_LSHIFT ||
		sdl.GetModState()&sdl.KMOD_RSHIFT == sdl.KMOD_RSHIFT {
		return userinput.KeyModShift
	} else if sdl.GetModState()&sdl.KMOD_LCTRL == sdl.KMOD_LCTRL ||
		sdl.GetModState()&sdl.KMOD_RCTRL == sdl.KMOD_RCTRL {
		return userinput.KeyModCtrl
	}
	return userinput.KeyModNone
}
