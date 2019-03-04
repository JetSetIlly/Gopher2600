package sdltv

import (
	"gopher2600/television"

	"github.com/veandco/go-sdl2/sdl"
)

// guiLoop listens for SDL events and is run concurrently
func (tv *SDLTV) guiLoop() {
	for {
		ev := sdl.WaitEvent()
		switch ev := ev.(type) {

		// close window
		case *sdl.QuitEvent:
			tv.SetFeature(television.ReqSetVisibility, false)
			tv.onWindowClose.dispatch()

		case *sdl.KeyboardEvent:
			if ev.Type == sdl.KEYDOWN {
				tv.crit.guiMutex.Lock()
				tv.crit.keypress = ev.Keysym.Sym
				tv.crit.guiMutex.Unlock()
				tv.onKeyboard.dispatch()
			}

		case *sdl.MouseButtonEvent:
			if ev.Type == sdl.MOUSEBUTTONDOWN {
				switch ev.Button {

				case sdl.BUTTON_LEFT:
					tv.onMouseButtonLeft.dispatch()

				case sdl.BUTTON_RIGHT:
					tv.crit.guiMutex.Lock()
					sx, sy := tv.scr.renderer.GetScale()

					// convert X pixel value to horizpos equivalent
					// the opposite of pixelX() and also the scalining applied
					// by the SDL renderer
					if tv.scr.unmasked {
						tv.crit.lastMouseHorizPos = int(float32(ev.X)/sx) - tv.Spec.ClocksPerHblank
					} else {
						tv.crit.lastMouseHorizPos = int(float32(ev.X) / sx)
					}

					// convert Y pixel value to scanline equivalent
					// the opposite of pixelY() and also the scalining applied
					// by the SDL renderer
					if tv.scr.unmasked {
						tv.crit.lastMouseScanline = int(float32(ev.Y) / sy)
					} else {
						tv.crit.lastMouseScanline = int(float32(ev.Y)/sy) + int(tv.scr.stb.visibleTopReference)
					}
					tv.crit.guiMutex.Unlock()

					tv.onMouseButtonRight.dispatch()
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
