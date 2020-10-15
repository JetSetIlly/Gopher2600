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

package sdldebug

import (
	"time"

	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/television"

	"github.com/veandco/go-sdl2/sdl"
)

func setupService() {
	// MOUSEMOTION events fill up the event queue pretty quickly. these take
	// time to service and for no good reason; we only want one value per frame
	// which we can do with a single call to GetMouseState()
	sdl.EventState(sdl.MOUSEMOTION, sdl.IGNORE)
}

// Service implements GuiCreator interface.
func (scr *SdlDebug) Service() {
	// run any outstanding feature requests
	select {
	case r := <-scr.featureReq:
		scr.serviceFeatureRequests(r)
	default:
	}

	// do not check for events if no event channel has been set
	if scr.events != nil {

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
				scr.showWindow(false)

			case *sdl.KeyboardEvent:
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
						case scr.events <- gui.EventKeyboard{
							Key:  sdl.GetKeyName(ev.Keysym.Sym),
							Mod:  mod,
							Down: true}:
						default:
							logger.Log("sdldebug", "dropped key down event")
						}
					}
				case sdl.KEYUP:
					if ev.Repeat == 0 {
						select {
						case scr.events <- gui.EventKeyboard{
							Key:  sdl.GetKeyName(ev.Keysym.Sym),
							Mod:  mod,
							Down: false}:
						default:
							logger.Log("sdldebug", "dropped key up event")
						}
					}
				}

			case *sdl.MouseButtonEvent:
				// the button event to send
				var button gui.MouseButton

				// mouse events are swallowed by the service loop
				// if they've been handled
				var swallow bool

				// in some contexts a debugging mouse event will be sent across
				// the events channel rather than a regular mouse event
				var debugClick bool

				switch ev.Button {
				case sdl.BUTTON_LEFT:
					button = gui.MouseButtonLeft

					// left mouse button should capture mouse if
					// not already done so.
					if !scr.isCaptured {
						swallow = true
						scr.isCaptured = true
						err := sdl.CaptureMouse(true)
						if err == nil {
							scr.window.SetGrab(true)
							sdl.ShowCursor(sdl.DISABLE)
							scr.window.SetTitle(windowTitleCaptured)
						}
					}

				case sdl.BUTTON_RIGHT:
					button = gui.MouseButtonRight

					// eight mouse button releases a captured mouse
					if scr.isCaptured {
						swallow = true
						scr.isCaptured = false
						err := sdl.CaptureMouse(false)
						if err == nil {
							scr.window.SetGrab(false)
							sdl.ShowCursor(sdl.ENABLE)
							scr.window.SetTitle(windowTitle)
						}
					} else {
						// if mouse is not captured then a right mouse
						// click is a debugging mouse click
						debugClick = true
					}
				}

				if !swallow {
					if debugClick {
						hp, sl := scr.convertMouseCoords(ev)
						select {
						case scr.events <- gui.EventDbgMouseButton{
							Button:   button,
							Down:     ev.Type == sdl.MOUSEBUTTONDOWN,
							X:        int(ev.X),
							Y:        int(ev.Y),
							HorizPos: hp,
							Scanline: sl}:
						default:
							if ev.Type == sdl.MOUSEBUTTONDOWN {
								logger.Log("sdlimgui", "dropped mouse down event")
							} else {
								logger.Log("sdlimgui", "dropped mouse up event")
							}
						}
					} else {
						select {
						case scr.events <- gui.EventMouseButton{
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
				}

			}
		}

		// mouse motion
		if scr.isCaptured {
			mx, my, _ := sdl.GetMouseState()
			if mx != scr.mx || my != scr.my {
				w, h := scr.window.GetSize()

				// reduce mouse x and y coordintes to the range 0.0 to 1.0
				//  no need to worry about negative numbers and numbers greater
				//  than 1.0 because we (should) have restricted mouse movement
				//  to the window (with window.SetGrab(). see the ReqCaptureMouse
				//  case in the ReqFeature() function)
				x := float32(mx) / float32(w)
				y := float32(my) / float32(h)

				select {
				case scr.events <- gui.EventMouseMotion{X: x, Y: y}:
				default:
					logger.Log("sdldebug", "dropped mouse motion event")
				}
				scr.mx = mx
				scr.my = my
			}
		}
	}

	// run any outstanding service functions
	select {
	case f := <-scr.service:
		f()
	default:
	}

	// sleep to help avoid 100% CPU usage when emulation is not running
	if scr.paused {
		<-time.After(time.Millisecond * 25)
	}
}

func (scr *SdlDebug) convertMouseCoords(ev *sdl.MouseButtonEvent) (int, int) {
	var hp, sl int

	sx, sy := scr.renderer.GetScale()

	// convert X pixel value to horizpos equivalent
	// the opposite of pixelX() and also the scalining applied
	// by the SDL renderer
	if scr.cropped {
		hp = int(float32(ev.X) / sx)

	} else {
		hp = int(float32(ev.X)/sx) - television.HorizClksHBlank
	}

	// convert Y pixel value to scanline equivalent
	// the opposite of pixelY() and also the scalining applied
	// by the SDL renderer
	if scr.cropped {
		sl = int(float32(ev.Y)/sy) + scr.topScanline
	} else {
		sl = int(float32(ev.Y) / sy)
	}

	return hp, sl
}
