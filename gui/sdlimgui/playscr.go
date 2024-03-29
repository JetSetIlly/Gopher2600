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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// note that values from the lazy package will not be updated in the service
// loop when the emulator is in playmode. nothing in winPlayScr() therefore
// should rely on any lazy value.

type playScr struct {
	img *SdlImgui

	// reference to screen data
	scr *screen

	// textures. displayTexture is the presentation texture
	displayTexture texture

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
	}

	// set texture, creation of textures will be done after every call to resize()
	// clamp is important for LINEAR filtering. not noticeable for NEAREST filtering
	win.displayTexture = img.rnd.addTexture(texturePlayscr, true, true)

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
	dl.AddImage(imgui.TextureID(win.displayTexture.getID()), win.imagePosMin, win.imagePosMax)

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
		fps, hz := win.img.dbg.VCS().TV.GetActualFPS()
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

	if coproc := win.img.cache.VCS.Mem.Cart.GetCoProc(); coproc != nil {
		clk := float32(win.img.dbg.VCS().Env.Prefs.ARM.Clock.Get().(float64))
		imgui.Text(fmt.Sprintf("%s Clock: %.0f Mhz", coproc.ProcessorID(), clk))
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
	if win.scr.nudgeIconCt > 0 {
		imgui.SameLine()
		imgui.Text(string(fonts.Nudge))
	}

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
	win.displayTexture.markForCreation()
	win.setScaling()
}

// render() implements the textureRenderer interface.
//
// render is called by service loop (via screen.render()). acquires it's own
// crit.section lock.
func (win *playScr) render() {
	win.scr.crit.section.Lock()
	defer win.scr.crit.section.Unlock()

	win.displayTexture.render(win.scr.crit.cropPixels)

	// unlike dbgscr, there is no need to call setScaling() every render()
}

// must be called from with a critical section.
func (win *playScr) setScaling() {
	rot := win.scr.rotation.Load().(specification.Rotation)

	sz := win.img.plt.displaySize()
	screenRegion := imgui.Vec2{X: sz[0], Y: sz[1]}

	w := float32(win.scr.crit.cropPixels.Bounds().Size().X)
	h := float32(win.scr.crit.cropPixels.Bounds().Size().Y)

	adj := float32(specification.AspectBias)
	if rot == specification.NormalRotation || rot == specification.FlippedRotation {
		adj *= pixelWidth
	}

	adjW := w * adj

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
	win.xscaling = scaling * adj
	win.scaledWidth = w * win.xscaling
	win.scaledHeight = h * win.yscaling

	// get visibleScanlines while we're in critical section
	win.visibleScanlines = win.scr.crit.frameInfo.VisibleBottom - win.scr.crit.frameInfo.VisibleTop
}

// textureSpec implements the scalingImage specification
func (win *playScr) textureSpec() (uint32, float32, float32) {
	return win.displayTexture.getID(), win.scaledWidth, win.scaledHeight
}
