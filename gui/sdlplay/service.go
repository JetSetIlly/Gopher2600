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

// Service implements gui.GUI interface.
//
// MUST only be called from the #mainthread
func (scr *SdlPlay) Service() {
	scr.lmtr.Wait()

	// run any outstanding service functions
	select {
	case f := <-scr.service:
		f()
	default:
	}

	// check for SDL events. timing out straight away if there's nothing
	sdlEvent := sdl.WaitEventTimeout(1)

	switch sdlEvent := sdlEvent.(type) {

	// close window
	case *sdl.QuitEvent:
		scr.eventChannel <- gui.Event{ID: gui.EventWindowClose}

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

		switch sdlEvent.Type {
		case sdl.KEYDOWN:
			if sdlEvent.Repeat == 0 {
				scr.eventChannel <- gui.Event{
					ID: gui.EventKeyboard,
					Data: gui.EventDataKeyboard{
						Key:  sdl.GetKeyName(sdlEvent.Keysym.Sym),
						Mod:  mod,
						Down: true}}
			}
		case sdl.KEYUP:
			if sdlEvent.Repeat == 0 {
				scr.eventChannel <- gui.Event{
					ID: gui.EventKeyboard,
					Data: gui.EventDataKeyboard{
						Key:  sdl.GetKeyName(sdlEvent.Keysym.Sym),
						Mod:  mod,
						Down: false}}
			}
		}
	}
}
