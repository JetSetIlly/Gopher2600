// This file is part of Gopher2600.
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
	"github.com/jetsetilly/imgui-go/v5"
)

func (win *winPrefs) drawCRT() {
	imgui.Spacing()

	// disable all CRT effect options if pixel-perfect is on
	imgui.PushItemWidth(-1)
	pixPerf := win.drawPixelPerfect()
	imgui.PopItemWidth()

	drawDisabled(pixPerf, func() {
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()

		if imgui.BeginTableV("crtprefs", 3, imgui.TableFlagsBordersInnerV, imgui.Vec2{}, 1.0) {
			imgui.TableSetupColumnV("0", imgui.TableColumnFlagsWidthFixed, 250, 0)
			imgui.TableSetupColumnV("1", imgui.TableColumnFlagsWidthFixed, 200, 1)
			imgui.TableSetupColumnV("2", imgui.TableColumnFlagsWidthFixed, 200, 2)

			imgui.TableNextRow()

			imgui.TableNextColumn()
			imgui.PushItemWidth(-1)

			usingBevel := win.drawUsingBevel()
			imgui.Spacing()
			drawDisabled(!usingBevel, win.drawEnvironment)
			imgui.Spacing()
			drawDisabled(!usingBevel, win.drawCurve)
			imgui.Spacing()
			drawDisabled(!usingBevel, win.drawRoundedCorners)

			imgui.PopItemWidth()
			imgui.Spacing()
			imgui.TableNextColumn()
			imgui.PushItemWidth(-1)

			win.drawInterference()
			imgui.Spacing()
			win.drawPhosphor()
			imgui.Spacing()
			win.drawMask()
			imgui.Spacing()
			win.drawScanlines()

			imgui.PopItemWidth()
			imgui.Spacing()
			imgui.TableNextColumn()
			imgui.PushItemWidth(-1)

			win.drawSharpness()
			imgui.Spacing()
			win.drawChromaticAberration()
			imgui.Spacing()
			win.drawBlackLevel()
			imgui.Spacing()
			win.drawShine()

			imgui.PopItemWidth()
			imgui.Spacing()
			imgui.EndTable()
		}
	})
}

func (win *winPrefs) drawUsingBevel() bool {
	bvl := win.img.crt.useBevel.Get().(bool)
	if imgui.Checkbox("Use Bevel##bevel", &bvl) {
		win.img.crt.useBevel.Set(bvl)
	}
	imgui.PushFont(win.img.fonts.smallGui)
	imgui.PushTextWrapPos()
	imgui.Text("There is currently only one bevel available. Future versions will allow a wider selection")
	imgui.PopTextWrapPos()
	imgui.PopFont()
	imgui.Spacing()
	imgui.Spacing()
	return bvl
}

func (win *winPrefs) drawCurve() {
	b := win.img.crt.curve.Get().(bool)
	if imgui.Checkbox("Curve##curve", &b) {
		win.img.crt.curve.Set(b)
	}

	f := float32(win.img.crt.curveAmount.Get().(float64))

	var label string

	if f >= 0.75 {
		label = "flat"
	} else if f >= 0.25 {
		label = "a little curved"
	} else {
		label = "very curved"
	}

	if imgui.SliderFloatV("##curveamount", &f, 1.0, -0.5, label, 1.0) {
		win.img.crt.curveAmount.Set(f)
	}
}

func (win *winPrefs) drawEnvironment() {
	b := win.img.crt.ambientTint.Get().(bool)
	if imgui.Checkbox("Blue Ambient Light##blueambientlight", &b) {
		win.img.crt.ambientTint.Set(b)
	}

	f := float32(win.img.crt.ambientTintStrength.Get().(float64))
	if imgui.SliderFloatV("##blueambientlightstrength", &f, 0.05, 0.6, "%0.2f", 1.0) {
		win.img.crt.ambientTintStrength.Set(f)
	}
}

func (win *winPrefs) drawMask() {
	b := win.img.crt.mask.Get().(bool)
	if imgui.Checkbox("Shadow Mask##mask", &b) {
		win.img.crt.mask.Set(b)
	}

	f := float32(win.img.crt.maskIntensity.Get().(float64))

	var label string

	if f >= 0.1 {
		label = "very visible"
	} else if f >= 0.075 {
		label = "visible"
	} else if f >= 0.05 {
		label = "faint"
	} else {
		label = "very faint"
	}

	if imgui.SliderFloatV("##maskintensity", &f, 0.025, 0.125, label, 1.0) {
		win.img.crt.maskIntensity.Set(f)
	}
}

func (win *winPrefs) drawScanlines() {
	b := win.img.crt.scanlines.Get().(bool)
	if imgui.Checkbox("Scanlines##scanlines", &b) {
		win.img.crt.scanlines.Set(b)
	}

	f := float32(win.img.crt.scanlinesIntensity.Get().(float64))

	var label string

	if f > 0.1 {
		label = "very visible"
	} else if f > 0.075 {
		label = "visible"
	} else if f >= 0.05 {
		label = "faint"
	} else {
		label = "very faint"
	}

	if imgui.SliderFloatV("##scanlinesintensity", &f, 0.025, 0.125, label, 1.0) {
		win.img.crt.scanlinesIntensity.Set(f)
	}
}

func (win *winPrefs) drawInterference() {
	b := win.img.crt.rfInterference.Get().(bool)
	if imgui.Checkbox("RF Noise / Ghosting##interference", &b) {
		win.img.crt.rfInterference.Set(b)
	}

	f := float32(win.img.crt.rfNoiseLevel.Get().(float64))

	var label string

	if f >= 0.1625 {
		label = "very high"
	} else if f >= 0.125 {
		label = "high"
	} else if f >= 0.0875 {
		label = "low"
	} else {
		label = "very low"
	}

	if imgui.SliderFloatV("##rfnoiselevel", &f, 0.05, 0.2, label, 1.0) {
		win.img.crt.rfNoiseLevel.Set(f)
	}

	f = float32(win.img.crt.rfGhostingLevel.Get().(float64))

	if f >= 0.1625 {
		label = "very high"
	} else if f >= 0.125 {
		label = "high"
	} else if f >= 0.0875 {
		label = "low"
	} else {
		label = "very low"
	}

	if imgui.SliderFloatV("##rfghostinglevel", &f, 0.05, 0.2, label, 1.0) {
		win.img.crt.rfGhostingLevel.Set(f)
	}
}

func (win *winPrefs) drawChromaticAberration() {
	imgui.Text("Chromatic Aberration")
	f := float32(win.img.crt.chromaticAberration.Get().(float64))

	var label string

	if f >= 1.5 {
		label = "very high"
	} else if f >= 1.00 {
		label = "high"
	} else if f >= 0.50 {
		label = "low"
	} else {
		label = "very low"
	}

	if imgui.SliderFloatV("##aberration", &f, 0.0, 0.2, label, 1.0) {
		win.img.crt.chromaticAberration.Set(f)
	}
}

func (win *winPrefs) drawPhosphor() {
	b := win.img.crt.phosphor.Get().(bool)
	if imgui.Checkbox("Phosphor##phosphor", &b) {
		win.img.crt.phosphor.Set(b)
	}

	var label string

	// latency
	f := float32(win.img.crt.phosphorLatency.Get().(float64))

	if f > 0.6 {
		label = "very slow"
	} else if f >= 0.4 {
		label = "slow"
	} else if f >= 0.2 {
		label = "fast"
	} else {
		label = "very fast"
	}

	if imgui.SliderFloatV("##phosphorlatency", &f, 0.8, 0.0, label, 1.0) {
		win.img.crt.phosphorLatency.Set(f)
	}

	// bloom
	g := float32(win.img.crt.phosphorBloom.Get().(float64))

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
		win.img.crt.phosphorBloom.Set(g)
	}
}

func (win *winPrefs) drawSharpness() {
	imgui.Text("Sharpness")

	f := float32(win.img.crt.sharpness.Get().(float64))

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
		win.img.crt.sharpness.Set(f)
	}
}

func (win *winPrefs) drawBlackLevel() {
	imgui.Text("Black Level")

	f := float32(win.img.crt.blackLevel.Get().(float64))

	var label string

	if f >= 0.08 {
		label = "very light"
	} else if f >= 0.04 {
		label = "light"
	} else {
		label = "dark"
	}

	if imgui.SliderFloatV("##blacklevel", &f, 0.00, 0.20, label, 1.0) {
		win.img.crt.blackLevel.Set(f)
	}
}

func (win *winPrefs) drawRoundedCorners() {
	b := win.img.crt.roundedCorners.Get().(bool)
	if imgui.Checkbox("Rounded Corners##roundedcorners", &b) {
		win.img.crt.roundedCorners.Set(b)
	}

	f := float32(win.img.crt.roundedCornersAmount.Get().(float64))

	var label string

	if f >= 0.07 {
		label = "extremely round"
	} else if f >= 0.05 {
		label = "very round"
	} else if f >= 0.03 {
		label = "quite rounded"
	} else {
		label = "hardly round at all"
	}

	if imgui.SliderFloatV("##roundedcornersamount", &f, 0.02, 0.09, label, 1.0) {
		win.img.crt.roundedCornersAmount.Set(f)
	}
}

func (win *winPrefs) drawShine() {
	b := win.img.crt.shine.Get().(bool)
	if imgui.Checkbox("Shine##shine", &b) {
		win.img.crt.shine.Set(b)
	}
}

func (win *winPrefs) drawPixelPerfect() bool {
	b := win.img.crt.pixelPerfect.Get().(bool)
	if imgui.Checkbox("Pixel Perfect##pixelpefect", &b) {
		win.img.crt.pixelPerfect.Set(b)
	}

	drawDisabled(!win.img.crt.pixelPerfect.Get().(bool), func() {
		imgui.SameLineV(0, 25)

		f := float32(win.img.crt.pixelPerfectFade.Get().(float64))

		var label string
		if f > 0.6 {
			label = "extreme fade"
		} else if f >= 0.4 {
			label = "high fade"
		} else if f > 0.2 {
			label = "tiny fade"
		} else if f == 0.0 {
			label = "no fade"
		}

		imgui.PushItemWidth(imguiRemainingWinWidth() * 0.75)
		if imgui.SliderFloatV("##pixelperfectfade", &f, 0.0, 0.8, label, 1.0) {
			win.img.crt.pixelPerfectFade.Set(f)
		}
		imgui.PopItemWidth()

		win.img.imguiTooltipSimple(`The fade slider controls how quickly pixels fade to
black. It is similar to the phosphor option that is
available when 'Pixel Perfect' mode is disabled.`)
	})

	return b
}
