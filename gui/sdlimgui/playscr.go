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
	"github.com/jetsetilly/gopher2600/gui/fonts"
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

	// textures. displayTexture is the presentation texture
	displayTexture uint32

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

	// fps overlay
	fpsPulse *time.Ticker
	fps      string
	hz       string

	// controller notifications
	peripheralLeft  peripheralNotification
	peripheralRight peripheralNotification

	// emulation notifications
	emulationNotice emulationEventNotification

	// cartridge notifications
	cartridgeNotice cartridgeEventNotification
}

func newPlayScr(img *SdlImgui) *playScr {
	win := &playScr{
		img:      img,
		scr:      img.screen,
		fpsPulse: time.NewTicker(time.Second),
		fps:      "waiting",
		peripheralRight: peripheralNotification{
			rightAlign: true,
		},
		emulationNotice: emulationEventNotification{
			emulation: img.dbg,
		},
	}

	// set texture, creation of textures will be done after every call to resize()
	gl.GenTextures(1, &win.displayTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.displayTexture)

	// mag and min changed in setScaling() according to whether we want pixel
	// perfect rendering
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)

	// clamp to edge is important for LINEAR filtering. not noticeable for
	// NEAREST filtering
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)

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
	dl.AddImage(imgui.TextureID(win.displayTexture), win.imagePosMin, win.imagePosMax)

	win.peripheralLeft.draw(win)
	win.peripheralRight.draw(win)
	win.cartridgeNotice.draw(win)

	if !win.drawFPS() {
		win.emulationNotice.draw(win, false)
	}
}

func (win *playScr) toggleFPS() {
	fps := win.img.prefs.fpsOverlay.Get().(bool)
	win.img.prefs.fpsOverlay.Set(!fps)
}

func (win *playScr) drawFPS() bool {
	if !win.img.prefs.fpsOverlay.Get().(bool) {
		return false
	}

	// update fps
	select {
	case <-win.fpsPulse.C:
		fps, hz := win.img.tv.GetActualFPS()
		win.fps = fmt.Sprintf("%03.2f fps", fps)
		win.hz = fmt.Sprintf("%03.2fhz", hz)
	default:
	}

	imgui.SetNextWindowPos(imgui.Vec2{0, 0})

	imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.Transparent)

	fpsOpen := true
	imgui.BeginV("##playscrfps", &fpsOpen, imgui.WindowFlagsAlwaysAutoResize|
		imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|
		imgui.WindowFlagsNoDecoration|imgui.WindowFlagsNoSavedSettings|
		imgui.WindowFlagsNoBringToFrontOnFocus)

	imgui.Text(fmt.Sprintf("Emulation: %s", win.fps))
	fr := imgui.CurrentIO().Framerate()
	if fr == 0.0 {
		imgui.Text("Rendering: waiting")
	} else {
		imgui.Text(fmt.Sprintf("Rendering: %03.2f fps", fr))
	}

	imguiSeparator()

	if win.img.lz.Cart.HasCoProcBus {
		clk := float32(win.img.vcs.Instance.Prefs.ARM.Clock.Get().(float64))
		imgui.Text(fmt.Sprintf("%s Clock: %.0f Mhz", win.img.lz.Cart.CoProcID, clk))
		imguiSeparator()
	}

	imgui.Text(fmt.Sprintf("%.1fx scaling", win.yscaling))
	imgui.Text(fmt.Sprintf("%d total scanlines", win.scr.crit.frameInfo.TotalScanlines))

	imguiSeparator()

	imgui.Text(win.img.screen.crit.frameInfo.Spec.ID)
	imgui.SameLine()
	imgui.Text(win.hz)
	if !win.scr.crit.frameInfo.VSync {
		imgui.SameLine()
		imgui.Text(string(fonts.NoVSYNC))
	}

	imguiSeparator()
	imgui.Text(fmt.Sprintf("%d frame input lag", win.scr.crit.frameQueueLen))

	// queueAnalysis := strings.Builder{}
	// for i := 0; i < win.scr.crit.frameQueueLen; i++ {
	// 	if i == win.scr.crit.renderIdx {
	// 		if i == win.scr.crit.plotIdx {
	// 			queueAnalysis.WriteRune('*')
	// 		} else {
	// 			queueAnalysis.WriteRune('r')
	// 		}
	// 	} else if i == win.scr.crit.plotIdx {
	// 		queueAnalysis.WriteRune('p')
	// 	} else {
	// 		queueAnalysis.WriteRune('.')
	// 	}
	// }
	// imgui.Text(queueAnalysis.String())

	// if win.img.screen.crit.frameInfo.IsAtariSafe() {
	// 	imguiSeparator()
	// 	imgui.Text("atari safe")
	// }

	imgui.PopStyleColorV(2)

	imgui.Spacing()
	win.emulationNotice.draw(win, true)

	imgui.End()

	return true
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

		// screen texture
		gl.BindTexture(gl.TEXTURE_2D, win.displayTexture)
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))

	} else {
		// previous versions had a check for whether the screen is stable. this
		// is wrong, we should always update the texture even when the screen
		// is "unstable"

		// screen texture
		gl.BindTexture(gl.TEXTURE_2D, win.displayTexture)
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
	if win.img.crtPrefs == nil || win.img.crtPrefs.IntegerScaling.Get().(bool) {
		scaling = float32(int(scaling))
	}

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
}

// textureSpec implements the scalingImage specification
func (win *playScr) textureSpec() (uint32, float32, float32) {
	return win.displayTexture, win.scaledWidth, win.scaledHeight
}
