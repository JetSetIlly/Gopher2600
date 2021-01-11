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

const winCRTPrefsTitle = "CRT Preferences"

type winCRTPrefs struct {
	img  *SdlImgui
	open bool

	// reference to screen data
	scr *screen

	// crt preview segment
	crtTexture uint32

	// height of the area containing the settings sliders
	settingsH float32
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

	return win, nil
}

func (win *winCRTPrefs) init() {
}

func (win *winCRTPrefs) destroy() {
}

func (win *winCRTPrefs) id() string {
	return winCRTPrefsTitle
}

func (win *winCRTPrefs) isOpen() bool {
	return win.open
}

func (win *winCRTPrefs) setOpen(open bool) {
	win.open = open
}

// height/width for detailPixels.
const (
	detailPixelsWidth  = 50
	detailPixelsHeight = 100
)

// the amount to adjust the pixel view to account for the HMOVE margin.
const HmoveMargin = 16

func (win *winCRTPrefs) draw() {
	if !win.open {
		return
	}

	win.scr.crit.section.Lock()

	// we're not too bothered about performance when the CRT prefs window is
	// open. figure out pixels and copy to texture every draw() frame.
	r := image.Rect(
		specification.HorizClksHBlank+HmoveMargin, win.scr.crit.topScanline,
		specification.HorizClksHBlank+HmoveMargin+detailPixelsWidth, win.scr.crit.topScanline+detailPixelsHeight,
	)
	pixels := win.scr.crit.pixels.SubImage(r).(*image.RGBA)

	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(pixels.Stride)/4)
	gl.BindTexture(gl.TEXTURE_2D, win.crtTexture)
	gl.TexImage2D(gl.TEXTURE_2D, 0,
		gl.RGBA, detailPixelsWidth, detailPixelsHeight, 0,
		gl.RGBA, gl.UNSIGNED_BYTE,
		gl.Ptr(pixels.Pix))

	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
	win.scr.crit.section.Unlock()

	imgui.SetNextWindowPosV(imgui.Vec2{10, 10}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winCRTPrefsTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	win.drawEnabled()
	imguiSeparator()

	// note start position of setting group
	win.settingsH = measureHeight(func() {
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
		imgui.Image(imgui.TextureID(win.crtTexture), imgui.Vec2{win.settingsH, win.settingsH})
		imgui.EndGroup()
	}
}
