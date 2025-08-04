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
	"github.com/jetsetilly/gopher2600/gui/display/bevels"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/imgui-go/v5"
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
	bevelTexture    texture
	bevelPosMin     imgui.Vec2
	bevelPosMax     imgui.Vec2
	bevelWidth      float32
	bevelHeight     float32
	bevelRatio      float32
	bevelRimTexture texture

	// whether the bevel is being used
	usingBevel bool

	// the tv screen has captured mouse input
	isCaptured bool

	// number of scanlines in current image. taken from screen but is crit section safe
	visibleScanlines int

	// overlays and subtitles for play screen
	overlay   playscrOverlay
	subtitles playscrSubtitles
}

func newPlayScr(img *SdlImgui) *playScr {
	win := &playScr{
		img: img,
		scr: img.screen,
		overlay: playscrOverlay{
			img: img,
		},
		subtitles: playscrSubtitles{
			img: img,
		},
	}
	win.overlay.playscr = win

	// set texture, creation of textures will be done after every call to resize()
	// clamp is important for LINEAR filtering. not noticeable for NEAREST filtering
	win.screenTexture = img.rnd.addTexture(shaderPlayscr, true, true, nil)
	win.bevelTexture = img.rnd.addTexture(shaderBevel, true, true, false)

	// additional configuration detail for the rim texture
	win.bevelRimTexture = img.rnd.addTexture(shaderBevel, true, true, true)

	// render bevel texture once on initlisation
	win.bevelTexture.render(bevels.SolidState.Bevel)
	win.bevelRimTexture.render(bevels.SolidState.BevelRim)

	// set scale and padding on startup. scale and padding will be recalculated
	// on window resize and textureRenderer.resize()
	win.scr.crit.section.Lock()
	win.resize()
	win.scr.crit.section.Unlock()

	return win
}

// the usingBevel flag is set in the draw() function however, we need to make
// sure it's set before a resize too. this is particularly important on
// initialisation when the resize() function is called before the draw()
// function is called fro the first time. without explicitely setting the
// usingBevel flag the scaleing of the TV screen will not take into account the
// BiasY value for the bevel
func (win *playScr) setUsingBevel() {
	// note whether we're using a bevel image or not
	win.usingBevel = win.img.rnd.supportsCRT() && !win.img.crt.pixelPerfect.Get().(bool) && win.img.crt.useBevel.Get().(bool)

	// rotation also plays a part in the decision to use the bevel
	win.usingBevel = win.usingBevel && win.img.screen.rotation.Load() == specification.NormalRotation
}

func (win *playScr) draw() {
	win.img.screen.crit.section.Lock()
	defer win.img.screen.crit.section.Unlock()

	win.setUsingBevel()

	dl := imgui.BackgroundDrawList()

	if win.usingBevel {
		dl.AddImage(imgui.TextureID(win.screenTexture.getID()), win.screenPosMin, win.screenPosMax)
		dl.AddImage(imgui.TextureID(win.bevelTexture.getID()), win.bevelPosMin, win.bevelPosMax)
		dl.AddImage(imgui.TextureID(win.bevelRimTexture.getID()), win.bevelPosMin, win.bevelPosMax)
	} else {
		dl.AddImage(imgui.TextureID(win.screenTexture.getID()), win.screenPosMin, win.screenPosMax)
	}

	if win.usingBevel {
		win.overlay.draw(win.bevelPosMin, win.bevelPosMax)
	} else {
		winw, winh := win.img.plt.windowSize()
		win.overlay.draw(imgui.Vec2{}, imgui.Vec2{X: winw, Y: winh})
	}
	win.subtitles.draw()
}

// resize() implements the textureRenderer interface.
//
// must be called from within a critical section.
func (win *playScr) resize() {
	// see comment for setUsingBevel() function for why we call it here
	win.setUsingBevel()

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

	// note that we don't need to render the bevel texture every frame because
	// it never changes

	// unlike dbgscr, there is also no need to call setScaling() every render()
}

// must be called from with a critical section.
func (win *playScr) setScalingBevel() {
	sz := bevels.SolidState.Bevel.Bounds().Size()
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
	if win.usingBevel {
		tvH *= bevels.SolidState.BiasY
	}

	winW, winH := win.img.plt.windowSize()
	winRatio := winW / winH

	// handle screen rotation
	rotation := win.scr.rotation.Load().(specification.Rotation)
	if rotation != specification.NormalRotation && rotation != specification.FlippedRotation {
		tvW, tvH = tvH, tvW
		tvW *= 1.05
	}

	tvRatio := tvW / tvH

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

// SetRotation implements the television.PixelRendererRotation interface
func (win *playScr) SetRotation(rotation specification.Rotation) {
	win.resize()
}
