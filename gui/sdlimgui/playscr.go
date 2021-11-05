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
	"fmt"
	"time"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
)

// note that values from the lazy package will not be updated in the service
// loop when the emulator is in playmode. nothing in winPlayScr() therefore
// should rely on any lazy value.

type playScr struct {
	img *SdlImgui

	// reference to screen data
	scr *screen

	// (re)create textures on next render()
	createTextures bool

	// textures
	screenTexture uint32

	// the tv screen has captured mouse input
	isCaptured bool

	imagePosMin imgui.Vec2
	imagePosMax imgui.Vec2

	// scaling of texture and calculated dimensions
	xscaling     float32
	yscaling     float32
	scaledWidth  float32
	scaledHeight float32

	// number of scanlines in current image. taken from screen but is crit section safe
	visibleScanlines int

	// fps
	fpsOpen  bool
	fpsPulse *time.Ticker
	fps      string
	hz       string

	// controller notifications
	peripheralLeft  peripheralNotification
	peripheralRight peripheralNotification

	// emulation events notifications
	emulationEvent emulationEventNotification

	// cartridge events notifications
	cartridgeEvent cartridgeEventNotification
}

func newPlayScr(img *SdlImgui) *playScr {
	win := &playScr{
		img:             img,
		scr:             img.screen,
		fpsPulse:        time.NewTicker(time.Second),
		fps:             "waiting",
		peripheralRight: peripheralNotification{rightAlign: true},
		emulationEvent:  emulationEventNotification{emulation: img.emulation},
	}

	// set texture, creation of textures will be done after every call to resize()
	gl.GenTextures(1, &win.screenTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.screenTexture)

	// mag and min changed in setScaling() according to whether we want pixel
	// perfect rendering
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	// clamp to edge is important for LINEAR filtering. not noticeable for
	// NEAREST filtering
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	// set scale and padding on startup. scale and padding will be recalculated
	// on window resize and textureRenderer.resize()
	win.scr.crit.section.Lock()
	win.setScaling()
	win.scr.crit.section.Unlock()

	return win
}

func (win *playScr) draw() {
	win.img.screen.crit.section.Lock()
	defer win.img.screen.crit.section.Unlock()

	dl := imgui.BackgroundDrawList()
	dl.AddImage(imgui.TextureID(win.screenTexture), win.imagePosMin, win.imagePosMax)

	if win.fpsOpen {
		// update fps
		select {
		case <-win.fpsPulse.C:
			fps, hz := win.img.tv.GetActualFPS()
			if win.scr.crit.frameInfo.VSynced {
				win.fps = fmt.Sprintf("%03.2f fps", fps)
			} else {
				win.fps = "unsynced"
			}
			win.hz = fmt.Sprintf("%03.2fhz", hz)
		default:
		}

		imgui.SetNextWindowPos(imgui.Vec2{0, 0})

		imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.Transparent)
		imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.Transparent)

		imgui.BeginV("##playscrfps", &win.fpsOpen, imgui.WindowFlagsAlwaysAutoResize|
			imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|
			imgui.WindowFlagsNoDecoration|imgui.WindowFlagsNoSavedSettings)

		imgui.Text(fmt.Sprintf("Emulation: %s", win.fps))
		if win.img.polling.measuredRenderingTime == 0.0 {
			imgui.Text("Rendering: waiting")
		} else {
			imgui.Text(fmt.Sprintf("Rendering: %03.2f fps", win.img.polling.measuredRenderingTime))
		}

		imguiSeparator()

		imgui.Text(fmt.Sprintf("%.1fx scaling", win.yscaling))
		imgui.Text(fmt.Sprintf("%d total scanlines", win.scr.crit.frameInfo.TotalScanlines))

		if win.img.screen.crit.frameInfo.IsAtariSafe() {
			imgui.Text("atari safe")
		}

		imguiSeparator()

		imgui.Text(win.img.screen.crit.frameInfo.Spec.ID)
		imgui.SameLine()
		imgui.Text(win.hz)

		imgui.PopStyleColorV(2)
		imgui.End()
	}

	win.peripheralLeft.draw(win)
	win.peripheralRight.draw(win)
	win.emulationEvent.draw(win)
	win.cartridgeEvent.draw(win)
}

// resize() implements the textureRenderer interface.
func (win *playScr) resize() {
	win.createTextures = true
	win.setScaling()
}

// render() implements the textureRenderer interface.
//
// render is called by service loop (via screen.render()). acquires it's own
// crit.section lock.
func (win *playScr) render() {
	win.scr.crit.section.Lock()
	defer win.scr.crit.section.Unlock()

	pixels := win.scr.crit.cropPixels

	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(pixels.Stride)/4)
	defer gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)

	if win.createTextures {
		win.createTextures = false

		// (re)create textures
		gl.BindTexture(gl.TEXTURE_2D, win.screenTexture)
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))
	} else {
		// previous versions had a check for whether the screen is stable. this
		// is wrong, we should always update the texture even when the screen
		// is "unstable"
		gl.BindTexture(gl.TEXTURE_2D, win.screenTexture)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))
	}

	// unlike dbgscr, there is no need to call setScaling() every render()
}

// must be called from with a critical section.
func (win *playScr) setScaling() {
	sz := win.img.plt.displaySize()
	screenRegion := imgui.Vec2{sz[0], sz[1]}

	w := float32(win.scr.crit.cropPixels.Bounds().Size().X)
	h := float32(win.scr.crit.cropPixels.Bounds().Size().Y)
	adjW := w * pixelWidth * win.scr.crit.frameInfo.Spec.AspectBias

	var scaling float32

	winRatio := screenRegion.X / screenRegion.Y
	aspectRatio := adjW / h

	if aspectRatio < winRatio {
		// window wider than TV screen
		scaling = screenRegion.Y / h
	} else {
		// TV screen wider than window
		scaling = screenRegion.X / adjW
	}

	// limit scaling to 1x
	if scaling < 1 {
		scaling = 1
	}

	// limit scaling to whole integers
	scaling = float32(int(scaling))

	win.imagePosMin = imgui.Vec2{
		X: float32(int((screenRegion.X - (adjW * scaling)) / 2)),
		Y: float32(int((screenRegion.Y - (h * scaling)) / 2)),
	}
	win.imagePosMax = screenRegion.Minus(win.imagePosMin)

	win.yscaling = scaling
	win.xscaling = scaling * pixelWidth * win.scr.crit.frameInfo.Spec.AspectBias
	win.scaledWidth = w * win.xscaling
	win.scaledHeight = h * win.yscaling

	// get visibleScanlines while we're in critical section
	win.visibleScanlines = win.scr.crit.frameInfo.VisibleBottom - win.scr.crit.frameInfo.VisibleTop

	gl.BindTexture(gl.TEXTURE_2D, win.screenTexture)
}
