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
	"image"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v3"
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
	crtTexture      uint32
	phosphorTexture uint32

	// (re)create textures on next render()
	createTextures bool

	// height of the area containing the settings sliders
	previewDim float32

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

	gl.ActiveTexture(gl.TEXTURE0)
	gl.GenTextures(1, &win.crtTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.crtTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	gl.ActiveTexture(gl.TEXTURE0 + phosphorTextureUnitPrefsCRT)
	gl.GenTextures(1, &win.phosphorTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.phosphorTexture)
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

	win.drawEnabled()
	imguiSeparator()

	// note start position of setting group
	win.previewDim = measureHeight(func() {
		imgui.BeginGroup()

		win.drawPhosphor()
		imgui.Spacing()

		win.drawMask()
		imgui.Spacing()

		win.drawScanlines()
		imgui.Spacing()

		win.drawNoise()
		imgui.Spacing()

		win.drawBlur()
		imgui.Spacing()

		win.drawVignette()
		imgui.Spacing()

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
	phosphor := win.scr.crit.phosphor.SubImage(win.previewRect).(*image.RGBA)

	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(pixels.Stride)/4)
	defer gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)

	if win.createTextures {
		win.createTextures = false

		// (re)create textures
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, win.crtTexture)
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, previewWidth, previewHeight, 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))

		gl.ActiveTexture(gl.TEXTURE0 + phosphorTextureUnitPrefsCRT)
		gl.BindTexture(gl.TEXTURE_2D, win.phosphorTexture)
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, previewWidth, previewHeight, 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(phosphor.Pix))
	} else {
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, win.crtTexture)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))

		gl.ActiveTexture(gl.TEXTURE0 + phosphorTextureUnitPrefsCRT)
		gl.BindTexture(gl.TEXTURE_2D, win.phosphorTexture)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(phosphor.Bounds().Size().X), int32(phosphor.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(phosphor.Pix))
	}
}

func (win *winCRTPrefs) drawEnabled() {
	b := win.img.crtPrefs.Enabled.Get().(bool)
	if imgui.Checkbox("Enabled##enabled", &b) {
		win.img.crtPrefs.Enabled.Set(b)
	}

	if !win.img.isPlaymode() {
		imgui.SameLine()
		imgui.Text("(when in playmode)")
	}
}

func (win *winCRTPrefs) drawPhosphor() {
	b := win.img.crtPrefs.Phosphor.Get().(bool)
	if imgui.Checkbox("Phosphor##phosphor", &b) {
		win.img.crtPrefs.Phosphor.Set(b)
	}

	f := float32(win.img.crtPrefs.PhosphorSpeed.Get().(float64))

	var label string

	if f > 1.25 {
		label = "very fast"
	} else if f > 1.0 {
		label = "fast"
	} else if f >= 0.75 {
		label = "slow"
	} else {
		label = "very slow"
	}

	if imgui.SliderFloatV("##phosphorspeed", &f, 0.5, 1.5, label, 1.0) {
		win.img.crtPrefs.PhosphorSpeed.Set(f)
	}
}

func (win *winCRTPrefs) drawMask() {
	b := win.img.crtPrefs.Mask.Get().(bool)
	if imgui.Checkbox("Shadow Mask##mask", &b) {
		win.img.crtPrefs.Mask.Set(b)
	}

	f := float32(win.img.crtPrefs.MaskBrightness.Get().(float64))

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

	if imgui.SliderFloatV("##maskbrightness", &f, 0.0, 1.0, label, 1.0) {
		win.img.crtPrefs.MaskBrightness.Set(f)
	}
}

func (win *winCRTPrefs) drawScanlines() {
	b := win.img.crtPrefs.Scanlines.Get().(bool)
	if imgui.Checkbox("Scanlines##scanlines", &b) {
		win.img.crtPrefs.Scanlines.Set(b)
	}

	f := float32(win.img.crtPrefs.ScanlinesBrightness.Get().(float64))

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

	if imgui.SliderFloatV("##scanlinesbrightness", &f, 0.0, 1.0, label, 1.0) {
		win.img.crtPrefs.ScanlinesBrightness.Set(f)
	}
}

func (win *winCRTPrefs) drawNoise() {
	b := win.img.crtPrefs.Noise.Get().(bool)
	if imgui.Checkbox("Noise##noise", &b) {
		win.img.crtPrefs.Noise.Set(b)
	}

	f := float32(win.img.crtPrefs.NoiseLevel.Get().(float64))

	var label string

	if f > 0.75 {
		label = "very high"
	} else if f > 0.50 {
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

func (win *winCRTPrefs) drawBlur() {
	b := win.img.crtPrefs.Blur.Get().(bool)
	if imgui.Checkbox("Blur##blur", &b) {
		win.img.crtPrefs.Blur.Set(b)
	}

	f := float32(win.img.crtPrefs.BlurLevel.Get().(float64))

	var label string

	if f > 0.45 {
		label = "very high"
	} else if f > 0.30 {
		label = "high"
	} else if f >= 0.15 {
		label = "low"
	} else {
		label = "very low"
	}

	if imgui.SliderFloatV("##Blurlevel", &f, 0.0, 0.6, label, 1.0) {
		win.img.crtPrefs.BlurLevel.Set(f)
	}
}

func (win *winCRTPrefs) drawVignette() {
	b := win.img.crtPrefs.Vignette.Get().(bool)
	if imgui.Checkbox("Vignette##vignette", &b) {
		win.img.crtPrefs.Vignette.Set(b)
	}
}

func (win *winCRTPrefs) drawDiskButtons() {
	if imgui.Button("Save") {
		err := win.img.crtPrefs.Save()
		if err != nil {
			logger.Log("sdlimgui", fmt.Sprintf("could not save crt settings: %v", err))
		}
	}

	imgui.SameLine()
	if imgui.Button("Restore") {
		err := win.img.crtPrefs.Load()
		if err != nil {
			logger.Log("sdlimgui", fmt.Sprintf("could not restore crt settings: %v", err))
		}
	}

	imgui.SameLine()
	if imgui.Button("SetDefaults") {
		win.img.crtPrefs.SetDefaults()
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

		imgui.ImageButton(imgui.TextureID(win.crtTexture), imgui.Vec2{win.previewDim, win.previewDim})
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
