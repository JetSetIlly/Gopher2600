package sdlplay

import (
	"gopher2600/gui"

	"github.com/veandco/go-sdl2/sdl"
)

// guiLoop listens for SDL events and is run concurrently
func (scr *SdlPlay) guiLoop() {
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

		default:
		}
	}
}
