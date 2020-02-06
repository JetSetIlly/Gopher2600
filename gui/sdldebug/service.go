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

package sdldebug

import (
	"gopher2600/gui"
	"gopher2600/television"
	"gopher2600/test"

	"github.com/veandco/go-sdl2/sdl"
)

func setupService() {
	// MOUSEMOTION events fill up the event queue pretty quickly. these take
	// time to service and for no good reason; we only want one value per frame
	// which we can do with a single call to GetMouseState()
	sdl.EventState(sdl.MOUSEMOTION, sdl.IGNORE)
}

// Service implements GuiCreator interface.
//
// MUST ONLY be called from the #mainthread
func (scr *SdlDebug) Service() {
	test.AssertMainThread()

	// do not check for events if no event channel has been set
	if scr.eventChannel != nil {

		// loop until there are no more events to retreive. this loop is
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
				scr.eventChannel <- gui.EventWindowClose{}

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
						scr.eventChannel <- gui.EventKeyboard{
							Key:  sdl.GetKeyName(ev.Keysym.Sym),
							Mod:  mod,
							Down: true}
					}
				case sdl.KEYUP:
					if ev.Repeat == 0 {
						scr.eventChannel <- gui.EventKeyboard{
							Key:  sdl.GetKeyName(ev.Keysym.Sym),
							Mod:  mod,
							Down: false}
					}
				}

			case *sdl.MouseButtonEvent:
				// what type of signal we send depends on the state of the
				// isCaptured flag
				if scr.isCaptured {
					// mouse is captured, send regular left/right mouse button
					// events
					button := gui.MouseButtonLeft
					if ev.Button == sdl.BUTTON_RIGHT {
						button = gui.MouseButtonRight
					}

					scr.eventChannel <- gui.EventMouseButton{
						Button: button,
						Down:   ev.Type == sdl.MOUSEBUTTONDOWN}
				} else {
					switch ev.Button {
					case sdl.BUTTON_LEFT:
						// send regular left button event even if mouse has not
						// been captured
						scr.eventChannel <- gui.EventMouseButton{
							Button: gui.MouseButtonLeft,
							Down:   ev.Type == sdl.MOUSEBUTTONDOWN}

					case sdl.BUTTON_RIGHT:
						// if right button is pressed when mouse is not captured,
						// send debugging signal
						hp, sl := scr.convertMouseCoords(ev)
						scr.eventChannel <- gui.EventDbgMouseButton{
							Button:   gui.MouseButtonRight,
							Down:     ev.Type == sdl.MOUSEBUTTONDOWN,
							X:        int(ev.X),
							Y:        int(ev.Y),
							HorizPos: hp,
							Scanline: sl}
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
				//  case in the SetFeature() function)
				x := float32(mx) / float32(w)
				y := float32(my) / float32(h)

				scr.eventChannel <- gui.EventMouseMotion{X: x, Y: y}
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
}

func (scr *SdlDebug) convertMouseCoords(ev *sdl.MouseButtonEvent) (int, int) {
	var hp, sl int

	sx, sy := scr.renderer.GetScale()

	// convert X pixel value to horizpos equivalent
	// the opposite of pixelX() and also the scalining applied
	// by the SDL renderer
	if scr.masked {
		hp = int(float32(ev.X) / sx)

	} else {
		hp = int(float32(ev.X)/sx) - television.HorizClksHBlank
	}

	// convert Y pixel value to scanline equivalent
	// the opposite of pixelY() and also the scalining applied
	// by the SDL renderer
	if scr.masked {
		sl = int(float32(ev.Y)/sy) + int(scr.topScanline)
	} else {
		sl = int(float32(ev.Y) / sy)
	}

	return hp, sl
}
