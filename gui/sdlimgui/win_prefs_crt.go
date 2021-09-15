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

	"github.com/inkyblackness/imgui-go/v4"
)

func (win *winPrefs) drawCRT() {
	// sliders to fill width of the window
	imgui.PushItemWidth(300)
	defer imgui.PopItemWidth()

	imgui.Spacing()
	pixPerf := win.drawPixelPerfect()
	imgui.Spacing()

	// disable all CRT effect options if pixel-perfect is on
	if pixPerf {
		imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
		imgui.PushStyleVarFloat(imgui.StyleVarAlpha, 0.5)
	}

	imgui.Separator()
	imgui.Spacing()

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

	win.drawGhosting()
	imgui.Spacing()

	win.drawPhosphor()
	win.drawSharpness()
	imgui.Spacing()

	win.drawUnsyncTolerance()
	imgui.Spacing()

	// end of disabling rule before drawing the disk buttons
	if pixPerf {
		imgui.PopStyleVar()
		imgui.PopItemFlag()
	}
}

func (win *winPrefs) drawCurve() {
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

func (win *winPrefs) drawMask() {
	b := win.img.crtPrefs.Mask.Get().(bool)
	if imgui.Checkbox("Shadow Mask##mask", &b) {
		win.img.crtPrefs.Mask.Set(b)
	}

	f := float32(win.img.crtPrefs.MaskBright.Get().(float64))

	var label string

	if f >= 1.0 {
		label = "very bright"
	} else if f >= 0.75 {
		label = "bright"
	} else if f >= 0.5 {
		label = "dark"
	} else {
		label = "very dark"
	}

	if imgui.SliderFloatV("##maskbright", &f, 0.25, 1.25, label, 1.0) {
		win.img.crtPrefs.MaskBright.Set(f)
	}

	fine := float32(win.img.crtPrefs.MaskFine.Get().(float64))

	if fine >= 3.0 {
		label = "very fine"
	} else if fine >= 2.5 {
		label = "fine"
	} else if fine >= 2.0 {
		label = "coarse"
	} else {
		label = "very coarse"
	}

	if imgui.SliderFloatV("##maskfine", &fine, 1.5, 3.5, label, 1.0) {
		win.img.crtPrefs.MaskFine.Set(fine)
	}
}

func (win *winPrefs) drawScanlines() {
	b := win.img.crtPrefs.Scanlines.Get().(bool)
	if imgui.Checkbox("Scanlines##scanlines", &b) {
		win.img.crtPrefs.Scanlines.Set(b)
	}

	f := float32(win.img.crtPrefs.ScanlinesBright.Get().(float64))

	var label string

	if f > 1.0 {
		label = "very bright"
	} else if f > 0.75 {
		label = "bright"
	} else if f >= 0.5 {
		label = "dark"
	} else {
		label = "very dark"
	}

	if imgui.SliderFloatV("##scanlinesbright", &f, 0.25, 1.25, label, 1.0) {
		win.img.crtPrefs.ScanlinesBright.Set(f)
	}

	fine := float32(win.img.crtPrefs.ScanlinesFine.Get().(float64))

	if fine > 2.25 {
		label = "very fine"
	} else if fine > 2.00 {
		label = "fine"
	} else if fine >= 1.75 {
		label = "coarse"
	} else {
		label = "very coarse"
	}

	if imgui.SliderFloatV("##scanlinesfine", &fine, 1.5, 2.5, label, 1.0) {
		win.img.crtPrefs.ScanlinesFine.Set(fine)
	}
}

func (win *winPrefs) drawNoise() {
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

func (win *winPrefs) drawFringing() {
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

func (win *winPrefs) drawGhosting() {
	b := win.img.crtPrefs.Ghosting.Get().(bool)
	if imgui.Checkbox("Ghosting##ghosting", &b) {
		win.img.crtPrefs.Ghosting.Set(b)
	}

	f := float32(win.img.crtPrefs.GhostingAmount.Get().(float64))

	var label string

	if f >= 3.5 {
		label = "very high"
	} else if f >= 2.5 {
		label = "high"
	} else if f >= 1.5 {
		label = "low"
	} else {
		label = "very low"
	}

	if imgui.SliderFloatV("##ghostingamount", &f, 0.0, 4.5, label, 1.0) {
		win.img.crtPrefs.GhostingAmount.Set(f)
	}
}

func (win *winPrefs) drawPhosphor() {
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

func (win *winPrefs) drawSharpness() {
	f := float32(win.img.crtPrefs.Sharpness.Get().(float64))

	var label string

	if f >= 0.9 {
		label = "very soft"
	} else if f >= 0.65 {
		label = "soft"
	} else if f >= 0.4 {
		label = "sharp"
	} else {
		label = "very sharp"
	}

	if imgui.SliderFloatV("##sharpness", &f, 0.1, 1.1, label, 1.0) {
		win.img.crtPrefs.Sharpness.Set(f)
	}
}

func (win *winPrefs) drawPixelPerfect() bool {
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

	return b
}

func (win *winPrefs) drawUnsyncTolerance() {
	imgui.Text("Screen Roll on lost VSYNC")

	t := int32(win.img.crtPrefs.UnsyncTolerance.Get().(int))
	var label string
	if t == 0 {
		label = "immediately"
	} else {
		label = fmt.Sprintf("after %d frames", t)
	}

	if imgui.SliderIntV("##unsyncTolerance", &t, 0, 5, label, 1.0) {
		win.img.crtPrefs.UnsyncTolerance.Set(t)
	}
}
