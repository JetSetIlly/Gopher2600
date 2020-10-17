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
	"time"

	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/logger"

	"github.com/inkyblackness/imgui-go/v2"
	"github.com/veandco/go-sdl2/sdl"
)

// Service implements GuiCreator interface.
func (img *SdlImgui) Service() {
	// run any outstanding feature requests
	select {
	case r := <-img.featureReq:
		img.serviceFeatureRequests(r)
	default:
	}

	// do not check for events if no event channel has been set
	if img.events != nil {
		// loop until there are no more events to retrieve. this loop is
		// intimately connected with the framelimiter below. what we don't want
		// to loop for too long servicing events. however:
		//
		// 1. servicing just one event per frame is not enough, queued events
		//    will take one frame on longer to resolve
		//
		// 2. limiting the number of events serviced per frame has the same
		//    problem for very long queues
		//
		// 3. truncating events is not wanted because we may miss important
		//    user input
		//
		// best solution is the poll loop
		for ev := sdl.PollEvent(); ev != nil; ev = sdl.PollEvent() {
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
						if ev.Repeat == 0 {
							select {
							case img.events <- gui.EventKeyboard{
								Key:  sdl.GetKeyName(ev.Keysym.Sym),
								Mod:  mod,
								Down: true}:
							default:
								logger.Log("sdlimgui", "dropped key down event")
							}
						}
					case sdl.KEYUP:
						if ev.Repeat == 0 {
							select {
							case img.events <- gui.EventKeyboard{
								Key:  sdl.GetKeyName(ev.Keysym.Sym),
								Mod:  mod,
								Down: false}:
							default:
								logger.Log("sdlimgui", "dropped key up event")
							}
						}
					}
				} else {
					switch ev.Type {
					case sdl.KEYDOWN:
						img.io.KeyPress(int(ev.Keysym.Scancode))
						img.updateKeyModifier()
					case sdl.KEYUP:
						img.io.KeyRelease(int(ev.Keysym.Scancode))
						img.updateKeyModifier()
					}
				}

			case *sdl.MouseButtonEvent:
				// the button event to send
				var button gui.MouseButton

				switch ev.Button {
				case sdl.BUTTON_LEFT:
					button = gui.MouseButtonLeft
					if !img.hasModal && img.isHovered() {
						img.setCapture(true)
						err := sdl.CaptureMouse(true)
						if err == nil {
							img.plt.window.SetGrab(true)
							_, err = sdl.ShowCursor(sdl.DISABLE)
							if err != nil {
								logger.Log("sdlimgui", err.Error())
							}
						}
					}

				case sdl.BUTTON_RIGHT:
					button = gui.MouseButtonRight

					// right mouse button releases a captured mouse
					if img.isCaptured() && ev.Type == sdl.MOUSEBUTTONUP {
						img.setCapture(false)
						err := sdl.CaptureMouse(false)
						if err == nil {
							img.plt.window.SetGrab(false)
							_, err = sdl.ShowCursor(sdl.ENABLE)
							if err != nil {
								logger.Log("sdlimgui", err.Error())
							}
						}
					}
				}

				if img.isCaptured() {
					select {
					case img.events <- gui.EventMouseButton{
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
			if mx != img.mx || my != img.my {
				w, h := img.plt.window.GetSize()

				// reduce mouse x and y coordintes to the range 0.0 to 1.0
				//  no need to worry about negative numbers and numbers greater
				//  than 1.0 because we (should) have restricted mouse movement
				//  to the window (with window.SetGrab(). see the ReqCaptureMouse
				//  case in the ReqFeature() function)
				x := float32(mx) / float32(w)
				y := float32(my) / float32(h)

				select {
				case img.events <- gui.EventMouseMotion{X: x, Y: y}:
				default:
					logger.Log("sdlimgui", "dropped mouse motion event")
				}
				img.mx = mx
				img.my = my
			}
		}
	}

	// refresh lazy values
	if !img.isPlaymode() {
		img.lz.Refresh()
	}

	// Signal start of a new frame
	img.plt.newFrame()
	imgui.NewFrame()

	// imgui commands
	img.draw()

	// Rendering
	imgui.Render() // This call only creates the draw data list. Actual rendering to framebuffer is done below.

	img.glsl.preRender()
	img.screen.render()
	img.glsl.render(img.plt.displaySize(), img.plt.framebufferSize(), imgui.RenderedDrawData())
	img.plt.postRender()

	// run any outstanding service functions
	select {
	case f := <-img.service:
		f()
	default:
	}

	// sleep to help avoid 100% CPU usage
	if !img.isPlaymode() {
		<-time.After(time.Millisecond * 25)
	}
}
