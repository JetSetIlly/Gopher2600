package sdltv

import (
	"github.com/veandco/go-sdl2/sdl"
)

// guiLoop listens for SDL events and is run concurrently. critical sections
// protected by tv.guiLoopLock
func (tv *SDLTV) guiLoop() {
	for true {
		ev := sdl.WaitEvent()
		switch ev := ev.(type) {

		// close window
		case *sdl.QuitEvent:
			// SetVisibility is outside of the critical section
			tv.SetVisibility(false)

			// *CRITICAL SECTION*
			// (R) tv.onWindowClose
			tv.guiLoopLock.Lock()
			tv.onWindowClose.dispatch()
			tv.guiLoopLock.Unlock()

		case *sdl.KeyboardEvent:
			if ev.Type == sdl.KEYDOWN {
				switch ev.Keysym.Sym {
				case sdl.K_BACKQUOTE:
					// *CRITICAL SECTION*
					// (W) tv.scr
					// (R) tv.playScr, tv.dbgScr
					tv.guiLoopLock.Lock()
					if tv.scr == tv.dbgScr {
						tv.scr = tv.playScr
					} else {
						tv.scr = tv.dbgScr
					}
					tv.setWindowSize(tv.scr.width, tv.scr.height)
					tv.guiLoopLock.Unlock()
				}
			}

		case *sdl.MouseButtonEvent:
			if ev.Type == sdl.MOUSEBUTTONDOWN {
				switch ev.Button {
				case sdl.BUTTON_LEFT:
					sx, sy := tv.renderer.GetScale()

					// *CRITICAL SECTION*
					// (W) mouseX, mouseY
					// (R) tv.scr, tv.dbgScr
					// (R) tv.onMouseButton1
					tv.guiLoopLock.Lock()

					// convert X pixel value to horizpos equivalent
					// the opposite of pixelX() and also the scalining applied
					// by the SDL renderer
					if tv.scr == tv.dbgScr {
						tv.mouseX = int(float32(ev.X)/sx) - tv.Spec.ClocksPerHblank
					} else {
						tv.mouseX = int(float32(ev.X) / sx)
					}

					// convert Y pixel value to scanline equivalent
					// the opposite of pixelY() and also the scalining applied
					// by the SDL renderer
					if tv.scr == tv.dbgScr {
						tv.mouseY = int(float32(ev.Y) / sy)
					} else {
						tv.mouseY = int(float32(ev.Y)/sy) + tv.Spec.ScanlinesPerVBlank
					}

					tv.onMouseButton1.dispatch()

					tv.guiLoopLock.Unlock()

				case sdl.BUTTON_RIGHT:
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
