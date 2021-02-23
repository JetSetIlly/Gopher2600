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
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/logger"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/veandco/go-sdl2/sdl"
)

// Service implements GuiCreator interface.
func (img *SdlImgui) Service() {
	// refresh lazy values
	if !img.isPlaymode() {
		img.lz.Refresh()
	}

	// poll for sdl event or timeout
	ev := img.polling.wait()

	// some events we want to service even if an event channel has not been
	// set. very few "modes" will not set a channel but we should be mindful of
	// those events that we handle entirely within this GUI implementation
	//
	// for those events that do require an event channel to be defined we wrap
	// them in select block and log the dropped event in the default case
	for ; ev != nil; ev = sdl.PollEvent() {
		switch ev := ev.(type) {
		case *sdl.QuitEvent:
			if !img.hasModal {
				select {
				case img.events <- gui.EventQuit{}:
				default:
					logger.Log("sdlimgui", "dropped quit event")
				}
			}

		case *sdl.WindowEvent:
			img.screen.crit.section.Lock()
			img.playScr.setScaling()
			img.screen.crit.section.Unlock()

		case *sdl.TextInputEvent:
			if img.hasModal || !img.isCaptured() {
				img.io.AddInputCharacters(string(ev.Text[:]))
			}

		case *sdl.KeyboardEvent:
			// handle keys that have special meaning for GUI
			if ev.Type == sdl.KEYUP && ev.Repeat == 0 {
				handled := true

				switch sdl.GetKeyName(ev.Keysym.Sym) {
				case "Escape":
					// works in debug and playmode
					img.setCapture(!img.isCaptured())

				case "`":
					// works only in debug mode
					if !img.isPlaymode() {
						if img.state == gui.StatePaused {
							img.term.pushCommand("RUN")
						} else {
							img.term.pushCommand("HALT")
						}
					}

				case "F9":
					if img.isPlaymode() {
						w := img.wm.windows[winTIARevisionsID]
						w.setOpen(!w.isOpen())
					}

				case "F10":
					if img.isPlaymode() {
						w := img.wm.windows[winCRTPrefsID]
						w.setOpen(!w.isOpen())
					}

				case "F11":
					if img.isPlaymode() {
						img.plt.setFullScreen(!img.plt.fullScreen)
					}

				case "F12":
					if img.isPlaymode() {
						img.playScr.fpsOpen = !img.playScr.fpsOpen
					}

				case "Pause":
					// TODO: flip between debug and playmodes

				default:
					handled = false
				}

				if handled {
					break // event switch
				}
			}

			// forward unhandled keypresses to registered events handler.
			// but only when gui is in playmode, has captured input and
			// there is no modal window.
			if !img.hasModal && (img.isPlaymode() || img.isCaptured()) {
				mod := gui.KeyModNone

				if sdl.GetModState()&sdl.KMOD_LALT == sdl.KMOD_LALT ||
					sdl.GetModState()&sdl.KMOD_RALT == sdl.KMOD_RALT {
					mod = gui.KeyModAlt
				} else if sdl.GetModState()&sdl.KMOD_LSHIFT == sdl.KMOD_LSHIFT ||
					sdl.GetModState()&sdl.KMOD_RSHIFT == sdl.KMOD_RSHIFT {
					mod = gui.KeyModShift
				} else if sdl.GetModState()&sdl.KMOD_LCTRL == sdl.KMOD_LCTRL ||
					sdl.GetModState()&sdl.KMOD_RCTRL == sdl.KMOD_RCTRL {
					mod = gui.KeyModCtrl
				}

				switch ev.Type {
				case sdl.KEYDOWN:
					fallthrough
				case sdl.KEYUP:
					if ev.Repeat == 0 {
						select {
						case img.events <- gui.EventKeyboard{
							GUI:  img,
							Key:  sdl.GetKeyName(ev.Keysym.Sym),
							Mod:  mod,
							Down: ev.Type == sdl.KEYDOWN}:
						default:
							logger.Log("sdlimgui", "dropped key up event")
						}
					}
				}

				break // event switch
			}

			// remaining keypresses forwarded to imgui io system
			switch ev.Type {
			case sdl.KEYDOWN:
				img.io.KeyPress(int(ev.Keysym.Scancode))
				img.updateKeyModifier()
			case sdl.KEYUP:
				img.io.KeyRelease(int(ev.Keysym.Scancode))
				img.updateKeyModifier()
			}

		case *sdl.MouseButtonEvent:
			// the button event to send
			var button gui.MouseButton

			switch ev.Button {
			case sdl.BUTTON_LEFT:
				button = gui.MouseButtonLeft

			case sdl.BUTTON_RIGHT:
				button = gui.MouseButtonRight

				if ev.Type == sdl.MOUSEBUTTONUP {
					if img.isCaptured() {
						img.setCapture(false)
					} else if img.isPlaymode() {
						img.setCapture(true)
					}
				}
			}

			if img.isCaptured() {
				select {
				case img.events <- gui.EventMouseButton{
					GUI:    img,
					Button: button,
					Down:   ev.Type == sdl.MOUSEBUTTONDOWN}:
				default:
					if ev.Type == sdl.MOUSEBUTTONDOWN {
						logger.Log("sdlimgui", "dropped mouse down event")
					} else {
						logger.Log("sdlimgui", "dropped mouse up event")
					}
				}
			}

			// trigger service wake in time for next Service() iteration.
			// without this, the results of the mouse button will not be
			// seen until the timeout (in the next iteration) has elapsed.
			//
			// eg. closing a window: the window will be drawn on *this*
			// frame and *this* mouse button press will be acknowledged.
			// next frame the window will not be drawn. however, the *next*
			// frame will sleep until the time out - *this* mouse button
			// event has been consumed. calling alert() ensures there is no
			// delay in drawing the *next* frame
			img.polling.alert()

		case *sdl.MouseWheelEvent:
			var deltaX, deltaY float32
			if ev.X > 0 {
				deltaX++
			} else if ev.X < 0 {
				deltaX--
			}
			if ev.Y > 0 {
				deltaY++
			} else if ev.Y < 0 {
				deltaY--
			}
			img.io.AddMouseWheelDelta(deltaX*2, deltaY*2)
		}
	}

	// mouse motion
	if img.isCaptured() {
		mx, my, _ := sdl.GetMouseState()
		if mx != img.mouseX || my != img.mouseY {
			w, h := img.plt.window.GetSize()

			// reduce mouse x and y coordintes to the range 0.0 to 1.0
			//  no need to worry about negative numbers and numbers greater
			//  than 1.0 because we (should) have restricted mouse movement
			//  to the window (with window.SetGrab(). see the ReqCaptureMouse
			//  case in the ReqFeature() function)
			x := float32(mx) / float32(w)
			y := float32(my) / float32(h)

			select {
			case img.events <- gui.EventMouseMotion{
				GUI: img,
				X:   x, Y: y,
			}:
			default:
				logger.Log("sdlimgui", "dropped mouse motion event")
			}
			img.mouseX = mx
			img.mouseY = my
		}
	}

	// start of a new frame
	img.plt.newFrame()
	imgui.NewFrame()

	// draw all windows according to debug/playmode
	img.draw()

	// rendering
	imgui.Render() // This call only creates the draw data list. Actual rendering to framebuffer is done below.
	img.glsl.preRender()
	img.screen.render()
	img.glsl.render()
	img.plt.postRender()
}
