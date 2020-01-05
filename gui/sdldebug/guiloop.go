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

	"github.com/veandco/go-sdl2/sdl"
)

// guiLoop listens for SDL events and is run concurrently
func (scr *SdlDebug) guiLoop() {
	for {
		sdlEvent := sdl.WaitEvent()
		switch sdlEvent := sdlEvent.(type) {

		// close window
		case *sdl.QuitEvent:
			scr.SetFeature(gui.ReqSetVisibility, false)
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

		case *sdl.MouseButtonEvent:
			hp, sl := scr.convertMouseCoords(sdlEvent)
			switch sdlEvent.Type {
			case sdl.MOUSEBUTTONDOWN:
				switch sdlEvent.Button {

				case sdl.BUTTON_LEFT:
					scr.eventChannel <- gui.Event{
						ID: gui.EventMouseLeft,
						Data: gui.EventDataMouse{
							Down:     true,
							X:        int(sdlEvent.X),
							Y:        int(sdlEvent.Y),
							HorizPos: hp,
							Scanline: sl}}

				case sdl.BUTTON_RIGHT:
					scr.eventChannel <- gui.Event{
						ID: gui.EventMouseRight,
						Data: gui.EventDataMouse{
							Down:     true,
							X:        int(sdlEvent.X),
							Y:        int(sdlEvent.Y),
							HorizPos: hp,
							Scanline: sl}}
				}

			case sdl.MOUSEBUTTONUP:
				switch sdlEvent.Button {

				case sdl.BUTTON_LEFT:
					scr.eventChannel <- gui.Event{
						ID: gui.EventMouseLeft,
						Data: gui.EventDataMouse{
							Down:     false,
							X:        int(sdlEvent.X),
							Y:        int(sdlEvent.Y),
							HorizPos: hp,
							Scanline: sl}}

				case sdl.BUTTON_RIGHT:
					scr.eventChannel <- gui.Event{
						ID: gui.EventMouseRight,
						Data: gui.EventDataMouse{
							Down:     false,
							X:        int(sdlEvent.X),
							Y:        int(sdlEvent.Y),
							HorizPos: hp,
							Scanline: sl}}
				}
			}

		case *sdl.MouseMotionEvent:
			// !!TODO: panning of zoomed image

		case *sdl.MouseWheelEvent:
			// !!TODO: zoom image

		default:
		}
	}
}

func (scr *SdlDebug) convertMouseCoords(sdlEvent *sdl.MouseButtonEvent) (int, int) {
	var hp, sl int

	sx, sy := scr.pxl.renderer.GetScale()

	// convert X pixel value to horizpos equivalent
	// the opposite of pixelX() and also the scalining applied
	// by the SDL renderer
	if scr.pxl.unmasked {
		hp = int(float32(sdlEvent.X)/sx) - television.HorizClksHBlank
	} else {
		hp = int(float32(sdlEvent.X) / sx)
	}

	// convert Y pixel value to scanline equivalent
	// the opposite of pixelY() and also the scalining applied
	// by the SDL renderer
	if scr.pxl.unmasked {
		sl = int(float32(sdlEvent.Y) / sy)
	} else {
		sl = int(float32(sdlEvent.Y)/sy) + int(scr.pxl.playTop)
	}

	return hp, sl
}
