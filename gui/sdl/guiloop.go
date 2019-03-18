package sdl

import (
	"gopher2600/gui"

	"github.com/veandco/go-sdl2/sdl"
)

// guiLoop listens for SDL events and is run concurrently
func (tv *GUI) guiLoop() {
	for {
		sdlEvent := sdl.WaitEvent()
		switch sdlEvent := sdlEvent.(type) {

		// close window
		case *sdl.QuitEvent:
			tv.SetFeature(gui.ReqSetVisibility, false)
			tv.eventChannel <- gui.Event{ID: gui.EventWindowClose}

		case *sdl.KeyboardEvent:
			switch sdlEvent.Type {
			case sdl.KEYDOWN:
				tv.eventChannel <- gui.Event{
					ID: gui.EventKeyboard,
					Data: gui.EventDataKeyboard{
						Key:  sdl.GetKeyName(sdlEvent.Keysym.Sym),
						Down: true}}
			case sdl.KEYUP:
				tv.eventChannel <- gui.Event{
					ID: gui.EventKeyboard,
					Data: gui.EventDataKeyboard{
						Key:  sdl.GetKeyName(sdlEvent.Keysym.Sym),
						Down: false}}
			}

		case *sdl.MouseButtonEvent:
			hp, sl := tv.convertMouseCoords(sdlEvent)
			switch sdlEvent.Type {
			case sdl.MOUSEBUTTONDOWN:
				switch sdlEvent.Button {

				case sdl.BUTTON_LEFT:
					tv.eventChannel <- gui.Event{
						ID: gui.EventMouseLeft,
						Data: gui.EventDataMouse{
							Down:     true,
							X:        int(sdlEvent.X),
							Y:        int(sdlEvent.Y),
							HorizPos: hp,
							Scanline: sl}}

				case sdl.BUTTON_RIGHT:
					tv.eventChannel <- gui.Event{
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
					tv.eventChannel <- gui.Event{
						ID: gui.EventMouseLeft,
						Data: gui.EventDataMouse{
							Down:     false,
							X:        int(sdlEvent.X),
							Y:        int(sdlEvent.Y),
							HorizPos: hp,
							Scanline: sl}}

				case sdl.BUTTON_RIGHT:
					tv.eventChannel <- gui.Event{
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
			// TODO: panning of zoomed image

		case *sdl.MouseWheelEvent:
			// TODO: zoom image

		default:
		}
	}
}

func (tv *GUI) convertMouseCoords(sdlEvent *sdl.MouseButtonEvent) (int, int) {
	var hp, sl int

	sx, sy := tv.scr.renderer.GetScale()

	// convert X pixel value to horizpos equivalent
	// the opposite of pixelX() and also the scalining applied
	// by the SDL renderer
	if tv.scr.unmasked {
		hp = int(float32(sdlEvent.X)/sx) - tv.Spec.ClocksPerHblank
	} else {
		hp = int(float32(sdlEvent.X) / sx)
	}

	// convert Y pixel value to scanline equivalent
	// the opposite of pixelY() and also the scalining applied
	// by the SDL renderer
	if tv.scr.unmasked {
		sl = int(float32(sdlEvent.Y) / sy)
	} else {
		sl = int(float32(sdlEvent.Y)/sy) + int(tv.scr.stb.visibleTopReference)
	}

	return hp, sl
}
