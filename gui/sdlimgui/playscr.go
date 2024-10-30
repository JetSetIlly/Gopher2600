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
	"github.com/jetsetilly/gopher2600/gui/display/bevels"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

type playScr struct {
	img *SdlImgui

	// reference to screen data
	scr *screen

	// screen
	screenTexture texture
	screenPosMin  imgui.Vec2
	screenPosMax  imgui.Vec2
	screenWidth   float32
	screenHeight  float32
	screenRatio   float32

	// scaling of texture and calculated dimensions
	scaling float32

	// bevel
	bevelTexture texture
	bevelPosMin  imgui.Vec2
	bevelPosMax  imgui.Vec2
	bevelWidth   float32
	bevelHeight  float32
	bevelRatio   float32

	// whether the bevel is being used
	usingBevel bool

	// the tv screen has captured mouse input
	isCaptured bool

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
			fpsPulse: time.NewTicker(1 * time.Millisecond),
			fps:      "waiting",
		},
	}
	win.overlay.playscr = win

	// set texture, creation of textures will be done after every call to resize()
	// clamp is important for LINEAR filtering. not noticeable for NEAREST filtering
	win.screenTexture = img.rnd.addTexture(texturePlayscr, true, true)
	win.bevelTexture = img.rnd.addTexture(textureBevel, true, true)

	// set scale and padding on startup. scale and padding will be recalculated
	// on window resize and textureRenderer.resize()
	win.scr.crit.section.Lock()
	win.resize()
	win.scr.crit.section.Unlock()

	return win
}

func (win *playScr) draw() {
	win.img.screen.crit.section.Lock()
	defer win.img.screen.crit.section.Unlock()

	// note whether we're using a bevel image or not
	win.usingBevel = win.img.displayPrefs.CRT.Enabled.Get().(bool)

	dl := imgui.BackgroundDrawList()
	if win.usingBevel {
		dl.AddImage(imgui.TextureID(win.bevelTexture.getID()), win.bevelPosMin, win.bevelPosMax)
	}
	dl.AddImage(imgui.TextureID(win.screenTexture.getID()), win.screenPosMin, win.screenPosMax)

	win.overlay.draw()
}

// resize() implements the textureRenderer interface.
//
// must be called from within a critical section.
func (win *playScr) resize() {
	win.screenTexture.markForCreation()
	win.bevelTexture.markForCreation()

	win.setScalingBevel()
	win.setScalingDisplay()

	// get visibleScanlines while we're in critical section
	win.visibleScanlines = win.scr.crit.frameInfo.VisibleBottom - win.scr.crit.frameInfo.VisibleTop
}

// updateRefreshRate() implements the textureRenderer interface.
func (win *playScr) updateRefreshRate() {
	win.overlay.updateRefreshRate()
}

// render() implements the textureRenderer interface.
//
// render is called by service loop (via screen.render()). acquires it's own
// crit.section lock.
func (win *playScr) render() {
	win.scr.crit.section.Lock()
	defer win.scr.crit.section.Unlock()

	win.screenTexture.render(win.scr.crit.cropPixels)
	win.bevelTexture.render(bevels.TV)

	// unlike dbgscr, there is no need to call setScaling() every render()
}

// must be called from with a critical section.
func (win *playScr) setScalingBevel() {
	sz := bevels.TV.Bounds().Size()
	bw := float32(sz.X)
	bh := float32(sz.Y)
	bRatio := bw / bh

	winW, winH := win.img.plt.windowSize()
	winRatio := winW / winH

	var scaling float32

	// place bevel in middle of window as best as we can
	if bRatio < winRatio {
		scaling = winH / bh
		win.bevelPosMin = imgui.Vec2{
			X: float32(int((winW - (bw * scaling)) / 2)),
			Y: 0,
		}
	} else {
		scaling = winW / bw
		win.bevelPosMin = imgui.Vec2{
			X: 0,
			Y: float32(int((winH - (bh * scaling)) / 2)),
		}
	}

	win.bevelPosMax = imgui.Vec2{
		X: winW - win.bevelPosMin.X,
		Y: winH - win.bevelPosMin.Y,
	}

	win.bevelWidth = bw * scaling
	win.bevelHeight = bh * scaling
	win.bevelRatio = win.bevelWidth / win.bevelHeight
}

// must be called from with a critical section.
func (win *playScr) setScalingDisplay() {
	tvW := float32(specification.WidthTV)
	tvH := float32(specification.HeightTV)
	tvRatio := tvW / tvH

	// handle screen rotation
	rotation := win.scr.rotation.Load().(specification.Rotation)
	if rotation != specification.NormalRotation && rotation != specification.FlippedRotation {
		tvW, tvH = tvH, tvW
	}

	winW, winH := win.img.plt.windowSize()
	winRatio := winW / winH

	// calculate required scaling
	if tvRatio < winRatio {
		// window wider than TV screen
		win.scaling = winH / tvH
	} else {
		// TV screen wider than window
		win.scaling = winW / tvW
	}

	// limit scaling to 1x
	if win.scaling < 1 {
		win.scaling = 1
	}

	// place display in middle of window as best as we can
	win.screenPosMin = imgui.Vec2{
		X: float32(int((winW - (tvW * win.scaling)) / 2)),
		Y: float32(int((winH - (tvH * win.scaling)) / 2)),
	}
	win.screenPosMax = imgui.Vec2{
		X: winW - win.screenPosMin.X,
		Y: winH - win.screenPosMin.Y,
	}

	win.screenWidth = tvW * win.scaling
	win.screenHeight = tvH * win.scaling
	win.screenRatio = win.screenWidth / win.screenHeight
}

// textureSpec implements the scalingImage specification
func (win *playScr) textureSpec() (uint32, float32, float32) {
	return win.screenTexture.getID(), win.screenWidth, win.screenHeight
}

// SetRotation implements the television.PixelRendererRotation interface
func (win *playScr) SetRotation(rotation specification.Rotation) {
	win.resize()
}
