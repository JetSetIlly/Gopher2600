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
	"image"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
)

const winCRTPrefsID = "CRT Preferences"

type winCRTPrefs struct {
	img  *SdlImgui
	open bool

	// reference to screen data
	scr *screen

	// crt preview segment
	crtTexture uint32

	// (re)create textures on next render()
	createTextures bool

	// height of the area containing the settings sliders
	previewHeight float32

	// mouse position when it is hovered over the preview
	previewMousePos imgui.Vec2

	// the window into the screen pixels
	previewRect image.Rectangle
	previewMin  image.Point
	previewMax  image.Point
}

func newWinCRTPrefs(img *SdlImgui) (window, error) {
	win := &winCRTPrefs{
		img: img,
		scr: img.screen,
	}

	gl.GenTextures(1, &win.crtTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.crtTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	return win, nil
}

func (win *winCRTPrefs) init() {
}

func (win *winCRTPrefs) id() string {
	return winCRTPrefsID
}

func (win *winCRTPrefs) isOpen() bool {
	return win.open
}

func (win *winCRTPrefs) setOpen(open bool) {
	win.open = open
}

// the amount to adjust the pixel view to account for the HMOVE margin.
const HmoveMargin = 16

func (win *winCRTPrefs) draw() {
	if !win.open {
		return
	}

	if win.img.isPlaymode() {
		imgui.SetNextWindowPosV(imgui.Vec2{25, 25}, imgui.ConditionAppearing, imgui.Vec2{0, 0})
		imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsNoSavedSettings|imgui.WindowFlagsAlwaysAutoResize)
	} else {
		imgui.SetNextWindowPosV(imgui.Vec2{25, 25}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
		imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)
	}

	// note start position of setting group
	win.previewHeight = imguiMeasureHeight(func() {
		imgui.BeginGroup()

		win.drawCurve()
		imgui.Spacing()

		win.drawMask()
		imgui.Spacing()

		win.drawScanlines()
		imgui.Spacing()

		win.drawNoise()
		imgui.Spacing()

		win.drawFringing()
		imgui.Spacing()

		win.drawPhosphor()

		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()

		win.drawPixelPerfect()

		imgui.EndGroup()
	})

	win.drawPreview()

	imguiSeparator()
	win.drawDiskButtons()

	imgui.End()
}

// height/width for preview image.
const (
	previewWidth  = 50
	previewHeight = 100
)

// resize() implements the textureRenderer interface.
func (win *winCRTPrefs) resize() {
	win.previewMin = image.Point{
		X: specification.ClksHBlank,
		Y: win.scr.crit.topScanline,
	}

	win.previewMax = image.Point{
		X: specification.ClksScanline - previewWidth,
		Y: win.scr.crit.bottomScanline - previewHeight,
	}

	// preview rect starts in the top left hand corner of the screen image
	win.previewRect = image.Rect(
		win.previewMin.X, win.previewMin.Y,
		win.previewMin.X+previewWidth, win.previewMin.Y+previewHeight,
	)

	win.createTextures = true
}

// render() implements the textureRenderer interface.
//
// render is called by service loop (via screen.render()). must be inside
// screen critical section.
func (win *winCRTPrefs) render() {
	if !win.open {
		return
	}

	pixels := win.scr.crit.pixels.SubImage(win.previewRect).(*image.RGBA)

	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(pixels.Stride)/4)
	defer gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)

	if win.createTextures {
		win.createTextures = false

		// (re)create textures
		gl.BindTexture(gl.TEXTURE_2D, win.crtTexture)
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, previewWidth, previewHeight, 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))
	} else {
		gl.BindTexture(gl.TEXTURE_2D, win.crtTexture)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))
	}
}

func (win *winCRTPrefs) drawCurve() {
	b := win.img.crtPrefs.Curve.Get().(bool)
	if imgui.Checkbox("Curve##curve", &b) {
		win.img.crtPrefs.Curve.Set(b)
	}

	f := float32(win.img.crtPrefs.CurveAmount.Get().(float64))

	var label string

	if f >= 0.75 {
		label = "flat"
	} else if f >= 0.25 {
		label = "a little curved"
	} else {
		label = "very curved"
	}

	if imgui.SliderFloatV("##curveamount", &f, 1.0, 0.0, label, 1.0) {
		win.img.crtPrefs.CurveAmount.Set(f)
	}
}

func (win *winCRTPrefs) drawMask() {
	b := win.img.crtPrefs.Mask.Get().(bool)
	if imgui.Checkbox("Shadow Mask##mask", &b) {
		win.img.crtPrefs.Mask.Set(b)
	}

	f := float32(win.img.crtPrefs.MaskBright.Get().(float64))

	var label string

	if f >= 0.75 {
		label = "very bright"
	} else if f >= 0.50 {
		label = "bright"
	} else if f >= 0.25 {
		label = "dark"
	} else {
		label = "very dark"
	}

	if imgui.SliderFloatV("##maskbright", &f, 0.0, 1.0, label, 1.0) {
		win.img.crtPrefs.MaskBright.Set(f)
	}
}

func (win *winCRTPrefs) drawScanlines() {
	b := win.img.crtPrefs.Scanlines.Get().(bool)
	if imgui.Checkbox("Scanlines##scanlines", &b) {
		win.img.crtPrefs.Scanlines.Set(b)
	}

	f := float32(win.img.crtPrefs.ScanlinesBright.Get().(float64))

	var label string

	if f > 0.75 {
		label = "very bright"
	} else if f > 0.50 {
		label = "bright"
	} else if f >= 0.25 {
		label = "dark"
	} else {
		label = "very dark"
	}

	if imgui.SliderFloatV("##scanlinesbright", &f, 0.0, 1.0, label, 1.0) {
		win.img.crtPrefs.ScanlinesBright.Set(f)
	}
}

func (win *winCRTPrefs) drawNoise() {
	b := win.img.crtPrefs.Noise.Get().(bool)
	if imgui.Checkbox("Noise##noise", &b) {
		win.img.crtPrefs.Noise.Set(b)
	}

	f := float32(win.img.crtPrefs.NoiseLevel.Get().(float64))

	var label string

	if f >= 0.75 {
		label = "very high"
	} else if f >= 0.50 {
		label = "high"
	} else if f >= 0.25 {
		label = "low"
	} else {
		label = "very low"
	}

	if imgui.SliderFloatV("##noiselevel", &f, 0.0, 1.0, label, 1.0) {
		win.img.crtPrefs.NoiseLevel.Set(f)
	}
}

func (win *winCRTPrefs) drawFringing() {
	b := win.img.crtPrefs.Fringing.Get().(bool)
	if imgui.Checkbox("Colour Fringing##fringing", &b) {
		win.img.crtPrefs.Fringing.Set(b)
	}

	f := float32(win.img.crtPrefs.FringingAmount.Get().(float64))

	var label string

	if f >= 0.45 {
		label = "very high"
	} else if f >= 0.30 {
		label = "high"
	} else if f >= 0.15 {
		label = "low"
	} else {
		label = "very low"
	}

	if imgui.SliderFloatV("##fringingamount", &f, 0.0, 0.6, label, 1.0) {
		win.img.crtPrefs.FringingAmount.Set(f)
	}
}

func (win *winCRTPrefs) drawDiskButtons() {
	if imgui.Button("Save") {
		err := win.img.crtPrefs.Save()
		if err != nil {
			logger.Logf("sdlimgui", "could not save crt settings: %v", err)
		}
	}

	imgui.SameLine()
	if imgui.Button("Restore") {
		err := win.img.crtPrefs.Load()
		if err != nil {
			logger.Logf("sdlimgui", "could not restore crt settings: %v", err)
		}
	}

	imgui.SameLine()
	if imgui.Button("SetDefaults") {
		win.img.crtPrefs.SetDefaults()
	}
}

func (win *winCRTPrefs) drawPhosphor() {
	b := win.img.crtPrefs.Phosphor.Get().(bool)
	if imgui.Checkbox("Phosphor##phosphor", &b) {
		win.img.crtPrefs.Phosphor.Set(b)
	}

	var label string

	// latency
	f := float32(win.img.crtPrefs.PhosphorLatency.Get().(float64))

	if f > 0.7 {
		label = "very slow"
	} else if f >= 0.5 {
		label = "slow"
	} else if f >= 0.3 {
		label = "fast"
	} else {
		label = "very fast"
	}

	if imgui.SliderFloatV("##phosphorlatency", &f, 0.9, 0.1, label, 1.0) {
		win.img.crtPrefs.PhosphorLatency.Set(f)
	}

	// bloom
	g := float32(win.img.crtPrefs.PhosphorBloom.Get().(float64))

	if g > 1.70 {
		label = "very high bloom"
	} else if g >= 1.2 {
		label = "high bloom"
	} else if g >= 0.70 {
		label = "low bloom"
	} else {
		label = "very low bloom"
	}

	if imgui.SliderFloatV("##phosphorbloom", &g, 0.20, 2.20, label, 1.0) {
		win.img.crtPrefs.PhosphorBloom.Set(g)
	}
}

func (win *winCRTPrefs) drawPixelPerfect() {
	b := !win.img.crtPrefs.Enabled.Get().(bool)
	if imgui.Checkbox("Pixel Perfect##pixelpefect", &b) {
		win.img.crtPrefs.Enabled.Set(!b)
	}

	var label string

	f := float32(win.img.crtPrefs.PixelPerfectFade.Get().(float64))

	if f > 0.7 {
		label = "extreme fade"
	} else if f >= 0.4 {
		label = "high fade"
	} else if f > 0.0 {
		label = "tiny fade"
	} else if f == 0.0 {
		label = "no fade"
	}

	if imgui.SliderFloatV("##pixelperfectfade", &f, 0.0, 0.9, label, 1.0) {
		win.img.crtPrefs.PixelPerfectFade.Set(f)
	}
}

func (win *winCRTPrefs) drawPreview() {
	imgui.SameLine()

	if !win.img.isPlaymode() {
		imgui.BeginGroup()

		// push style info for screen and overlay ImageButton(). we're using
		// ImageButton because an Image will not capture mouse events and pass them
		// to the parent window. this means that a click-drag on the screen/overlay
		// will move the window, which we don't want.
		imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.Transparent)
		imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.Transparent)
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.Transparent)
		imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{0.0, 0.0})

		imgui.ImageButton(imgui.TextureID(win.crtTexture), imgui.Vec2{win.previewHeight, win.previewHeight})
		if imgui.IsItemHovered() {
			p := imgui.MousePos()

			if imgui.IsMouseDragging(0, 0.0) {
				const sensitivity = 4.0

				// measure mouse distance moved
				diff := image.Point{X: int((win.previewMousePos.X - p.X) / sensitivity),
					Y: int((win.previewMousePos.Y - p.Y) / sensitivity)}

				// makre sure changes out of bounds
				if win.previewRect.Min.X+diff.X > win.previewMax.X {
					diff.X = 0
				} else if win.previewRect.Min.X+diff.X < win.previewMin.X {
					diff.X = 0
				}
				if win.previewRect.Min.Y+diff.Y > win.previewMax.Y {
					diff.Y = 0
				} else if win.previewRect.Min.Y+diff.Y < win.previewMin.Y {
					diff.Y = 0
				}

				// commit changes
				win.previewRect = win.previewRect.Add(diff)
			}

			// store mouse position for next measurement
			win.previewMousePos = p
		}

		// pop style info for screen and overlay textures
		imgui.PopStyleVar()
		imgui.PopStyleColorV(3)

		imgui.EndGroup()
	}
}

// unlike equivalient functions for winDbgScr and winPlayScr this does not need
// to be called from with a critical section.
func (win *winCRTPrefs) getScaledWidth() float32 {
	return float32(win.previewRect.Size().X) * win.getScaling(true)
}

// unlike equivalient functions for winDbgScr and winPlayScr this does not need
// to be called from with a critical section.
func (win *winCRTPrefs) getScaledHeight() float32 {
	return float32(win.previewRect.Size().Y) * win.getScaling(false)
}

func (win *winCRTPrefs) getScaling(horiz bool) float32 {
	const scaling = 2.0

	if horiz {
		return pixelWidth * win.scr.aspectBias * scaling
	}

	return scaling
}
