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

	"github.com/inkyblackness/imgui-go/v3"
	"github.com/veandco/go-sdl2/sdl"
)

// time periods in milliseconds that each mode sleeps for at the end of each
// service() call. this changes depending on whether we're in debug or play
// mode.
const (
	debugSleepPeriod = 50
	playSleepPeriod  = 10
	idleSleepPeriod  = 500
)

// Service implements GuiCreator interface.
func (img *SdlImgui) Service() {
	// run any outstanding service functions
	select {
	case f := <-img.service:
		f()
	default:
	}

	// the first SDL event is captured by the select blocks below. queued
	// events are captured by the PollEvent loop
	var ev sdl.Event

	// wait for an event or a timeout depending on the state of the emulation.
	// are we in playmode or are we in debugging mode.
	//
	// note that the only difference between the select blocks is the timeout
	// duration.
	if img.isPlaymode() {
		select {
		case <-img.servicePulsePlay.C: // timeout
		case <-img.serviceWake:
		case ev = <-img.plt.miniEvent:
		case r := <-img.featureSet:
			img.serviceSetFeature(r)
		case r := <-img.featureGet:
			img.serviceGetFeature(r)
		}
	} else {
		// refresh lazy values. we have to do this every service iteration
		// because otherwise we might miss some changes:
		//
		// the debugger.HasChanged flag is itself a product of the lazy refresh
		// process so we must have read it in order to know if any changes have
		// taken place! I suppose we could have two layers of lazy refresh but
		// that seems to be unduly complicated for what would be a small gain.
		//
		// moreover, it's possible to change the state of the emulation from
		// the GUI without going through the debugger, which makes the
		// HasChanged flag of limited use.
		img.lz.Refresh()

		if img.lz.Debugger.HasChanged || img.state == gui.StateRunning {
			select {
			case <-img.servicePulseDebug.C: // timeout
			case <-img.serviceWake:
			case ev = <-img.plt.miniEvent:
			case r := <-img.featureSet:
				img.serviceSetFeature(r)
			case r := <-img.featureGet:
				img.serviceGetFeature(r)
			}
		} else {
			select {
			case <-img.servicePulseIdle.C: // timeout
			case <-img.serviceWake:
			case ev = <-img.plt.miniEvent:
				if !img.isCaptured() && !img.isPlaymode() {
					// slow down mouse events unless we're in playmode or input has been "captured"
					switch ev.(type) {
					case *sdl.MouseMotionEvent:
						<-img.servicePulseDebug.C
					}
				}
			case r := <-img.featureSet:
				img.serviceSetFeature(r)
			case r := <-img.featureGet:
				img.serviceGetFeature(r)
			}
		}
	}

	// do not service SDL events if no event channel has been set
	if img.events != nil {
		for ; ev != nil; ev = sdl.PollEvent() {
			switch ev := ev.(type) {
			// close window
			case *sdl.QuitEvent:
				if !img.hasModal {
					select {
					case img.events <- gui.EventQuit{}:
					default:
						panic("quit event jammed: forcing quit (contact developer)")
					}
				}

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

					case "F11":
						if img.isPlaymode() {
							img.plt.toggleFullScreen()
						}

					case "F12":
						if img.isPlaymode() {
							img.wm.playScr.fps.open = !img.wm.playScr.fps.open
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
				if img.events != nil && !img.hasModal && (img.isPlaymode() || img.isCaptured()) {
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

					// right mouse button releases a captured mouse
					if img.isCaptured() && ev.Type == sdl.MOUSEBUTTONUP {
						img.setCapture(false)
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
				// event has been consumed. prodding serviceWake ensures there
				// is no delay in drawing the *next* frame
				//
				// pushing event inside a select/default block to prevent
				// channel deadlock
				select {
				case img.serviceWake <- true:
				default:
				}

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
	img.glsl.render(img.plt.displaySize(), img.plt.framebufferSize(), imgui.RenderedDrawData())
	img.plt.postRender()
}
