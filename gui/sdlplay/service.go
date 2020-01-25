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

package sdlplay

import (
	"gopher2600/gui"

	"github.com/veandco/go-sdl2/sdl"
)

func setupService() {
	// MOUSEMOTION events fill up the event queue pretty quickly. these take
	// time to service and for no good reason; we only want one value per frame
	// which we can do with a single call to GetMouseState()
	sdl.EventState(sdl.MOUSEMOTION, sdl.IGNORE)
}

// Service implements gui.GUI interface.
//
// MUST ONLY be called from the #mainthread
func (scr *SdlPlay) Service() {

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
		empty := false
		for !empty {

			// check for SDL events. timing out straight away if there's nothing
			ev := sdl.WaitEventTimeout(1)

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
				scr.eventChannel <- gui.EventMouseButton{
					Button: gui.MouseButtonLeft,
					Down:   ev.Type == sdl.MOUSEBUTTONDOWN}

			case nil:
				// if we have a nil value then the WaitEvent has timed out
				// and we can say that the event queue is empty
				empty = true
			}
		}

		// !!TODO: GetMouseState()
	}

	// wait for frame limiter
	scr.lmtr.Wait()

	// run any outstanding service functions
	select {
	case f := <-scr.service:
		f()
	default:
	}
}
