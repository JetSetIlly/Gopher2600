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
	"time"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

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

	// overlay for play screen
	overlay playscrOverlay
}

func newPlayScr(img *SdlImgui) *playScr {
	win := &playScr{
		img: img,
		scr: img.screen,
		overlay: playscrOverlay{
			fpsPulse: time.NewTicker(time.Second),
			fps:      "waiting",
		},
	}
	win.overlay.playscr = win

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

	win.overlay.draw()
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

// SetRotation implements the television.PixelRendererRotation interface
func (win *playScr) SetRotation(rotation specification.Rotation) {
	win.resize()
}
