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
	"github.com/inkyblackness/imgui-go/v4"
)

func (win *winPrefs) drawCRT() {
	imgui.Spacing()

	// disable all CRT effect options if pixel-perfect is on
	imgui.PushItemWidth(-1)
	pixPerf := win.drawPixelPerfect()
	imgui.PopItemWidth()
	if pixPerf {
		imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
		imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
		defer imgui.PopStyleVar()
		defer imgui.PopItemFlag()
	}

	// there is deliberately no option for IntegerScaling in the GUI. the
	// option exists in the prefs file but we're not exposing the option to the
	// end user

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	if imgui.BeginTableV("displayPrefs.CRT", 3, imgui.TableFlagsBordersInnerV, imgui.Vec2{}, 1.0) {
		imgui.TableSetupColumnV("0", imgui.TableColumnFlagsWidthFixed, 200, 0)
		imgui.TableSetupColumnV("1", imgui.TableColumnFlagsWidthFixed, 200, 1)
		imgui.TableSetupColumnV("2", imgui.TableColumnFlagsWidthFixed, 200, 2)

		imgui.TableNextRow()

		imgui.TableNextColumn()
		imgui.PushItemWidth(-1)
		win.drawCurve()
		imgui.Spacing()
		win.drawRoundedCorners()
		imgui.Spacing()
		win.drawMask()
		imgui.Spacing()
		win.drawScanlines()
		imgui.PopItemWidth()

		imgui.TableNextColumn()
		imgui.PushItemWidth(-1)
		win.drawInterference()
		imgui.Spacing()
		win.drawFlicker()
		imgui.Spacing()
		win.drawFringing()
		imgui.Spacing()
		win.drawGhosting()
		imgui.Spacing()
		win.drawSharpness()
		imgui.PopItemWidth()

		imgui.TableNextColumn()
		imgui.PushItemWidth(-1)
		win.drawPhosphor()
		imgui.Spacing()
		win.drawBlackLevel()
		imgui.Spacing()
		win.drawBevel()
		imgui.Spacing()
		win.drawShine()
		imgui.PopItemWidth()

		imgui.EndTable()
	}
}

func (win *winPrefs) drawCurve() {
	b := win.img.displayPrefs.CRT.Curve.Get().(bool)
	if imgui.Checkbox("Curve##curve", &b) {
		win.img.displayPrefs.CRT.Curve.Set(b)
	}

	f := float32(win.img.displayPrefs.CRT.CurveAmount.Get().(float64))

	var label string

	if f >= 0.75 {
		label = "flat"
	} else if f >= 0.25 {
		label = "a little curved"
	} else {
		label = "very curved"
	}

	if imgui.SliderFloatV("##curveamount", &f, 1.0, 0.0, label, 1.0) {
		win.img.displayPrefs.CRT.CurveAmount.Set(f)
	}
}

func (win *winPrefs) drawMask() {
	b := win.img.displayPrefs.CRT.Mask.Get().(bool)
	if imgui.Checkbox("Shadow Mask##mask", &b) {
		win.img.displayPrefs.CRT.Mask.Set(b)
	}

	f := float32(win.img.displayPrefs.CRT.MaskIntensity.Get().(float64))

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
		win.img.displayPrefs.CRT.MaskIntensity.Set(f)
	}

	fine := float32(win.img.displayPrefs.CRT.MaskFine.Get().(float64))

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
		win.img.displayPrefs.CRT.MaskFine.Set(fine)
	}
}

func (win *winPrefs) drawScanlines() {
	b := win.img.displayPrefs.CRT.Scanlines.Get().(bool)
	if imgui.Checkbox("Scanlines##scanlines", &b) {
		win.img.displayPrefs.CRT.Scanlines.Set(b)
	}

	f := float32(win.img.displayPrefs.CRT.ScanlinesIntensity.Get().(float64))

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
		win.img.displayPrefs.CRT.ScanlinesIntensity.Set(f)
	}

	fine := float32(win.img.displayPrefs.CRT.ScanlinesFine.Get().(float64))

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
		win.img.displayPrefs.CRT.ScanlinesFine.Set(fine)
	}
}

func (win *winPrefs) drawInterference() {
	b := win.img.displayPrefs.CRT.Interference.Get().(bool)
	if imgui.Checkbox("Interference##interference", &b) {
		win.img.displayPrefs.CRT.Interference.Set(b)
	}

	f := float32(win.img.displayPrefs.CRT.InterferenceLevel.Get().(float64))

	var label string

	if f >= 0.18 {
		label = "very high"
	} else if f >= 0.16 {
		label = "high"
	} else if f >= 0.14 {
		label = "low"
	} else {
		label = "very low"
	}

	if imgui.SliderFloatV("##interferencelevel", &f, 0.1, 0.2, label, 1.0) {
		win.img.displayPrefs.CRT.InterferenceLevel.Set(f)
	}
}

func (win *winPrefs) drawFlicker() {
	b := win.img.displayPrefs.CRT.Flicker.Get().(bool)
	if imgui.Checkbox("Flicker##flicker", &b) {
		win.img.displayPrefs.CRT.Flicker.Set(b)
	}

	f := float32(win.img.displayPrefs.CRT.FlickerLevel.Get().(float64))

	var label string

	if f >= 0.06 {
		label = "very high"
	} else if f >= 0.04 {
		label = "high"
	} else if f >= 0.03 {
		label = "low"
	} else {
		label = "very low"
	}

	if imgui.SliderFloatV("##flickerlevel", &f, 0.01, 0.06, label, 1.0) {
		win.img.displayPrefs.CRT.FlickerLevel.Set(f)
	}
}

func (win *winPrefs) drawFringing() {
	b := win.img.displayPrefs.CRT.Fringing.Get().(bool)
	if imgui.Checkbox("Colour Fringing##fringing", &b) {
		win.img.displayPrefs.CRT.Fringing.Set(b)
	}

	f := float32(win.img.displayPrefs.CRT.FringingAmount.Get().(float64))

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
		win.img.displayPrefs.CRT.FringingAmount.Set(f)
	}
}

func (win *winPrefs) drawGhosting() {
	b := win.img.displayPrefs.CRT.Ghosting.Get().(bool)
	if imgui.Checkbox("Ghosting##ghosting", &b) {
		win.img.displayPrefs.CRT.Ghosting.Set(b)
	}

	f := float32(win.img.displayPrefs.CRT.GhostingAmount.Get().(float64))

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
		win.img.displayPrefs.CRT.GhostingAmount.Set(f)
	}
}

func (win *winPrefs) drawPhosphor() {
	b := win.img.displayPrefs.CRT.Phosphor.Get().(bool)
	if imgui.Checkbox("Phosphor##phosphor", &b) {
		win.img.displayPrefs.CRT.Phosphor.Set(b)
	}

	var label string

	// latency
	f := float32(win.img.displayPrefs.CRT.PhosphorLatency.Get().(float64))

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
		win.img.displayPrefs.CRT.PhosphorLatency.Set(f)
	}

	// bloom
	g := float32(win.img.displayPrefs.CRT.PhosphorBloom.Get().(float64))

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
		win.img.displayPrefs.CRT.PhosphorBloom.Set(g)
	}
}

func (win *winPrefs) drawSharpness() {
	imgui.Text("Sharpness")

	f := float32(win.img.displayPrefs.CRT.Sharpness.Get().(float64))

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
		win.img.displayPrefs.CRT.Sharpness.Set(f)
	}
}

func (win *winPrefs) drawBlackLevel() {
	imgui.Text("Black Level")

	f := float32(win.img.displayPrefs.CRT.BlackLevel.Get().(float64))

	var label string

	if f >= 0.08 {
		label = "very light"
	} else if f >= 0.04 {
		label = "light"
	} else {
		label = "dark"
	}

	if imgui.SliderFloatV("##blacklevel", &f, 0.00, 0.20, label, 1.0) {
		win.img.displayPrefs.CRT.BlackLevel.Set(f)
	}
}

func (win *winPrefs) drawRoundedCorners() {
	b := win.img.displayPrefs.CRT.RoundedCorners.Get().(bool)
	if imgui.Checkbox("Rounded Corners##roundedcorners", &b) {
		win.img.displayPrefs.CRT.RoundedCorners.Set(b)
	}

	f := float32(win.img.displayPrefs.CRT.RoundedCornersAmount.Get().(float64))

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
		win.img.displayPrefs.CRT.RoundedCornersAmount.Set(f)
	}
}

func (win *winPrefs) drawBevel() {
	b := win.img.displayPrefs.CRT.Bevel.Get().(bool)
	if imgui.Checkbox("Bevel##bevel", &b) {
		win.img.displayPrefs.CRT.Bevel.Set(b)
	}

	f := float32(win.img.displayPrefs.CRT.BevelSize.Get().(float64))

	var label string

	if f >= 0.02 {
		label = "very deep"
	} else if f >= 0.015 {
		label = "deep"
	} else if f >= 0.01 {
		label = "shallow"
	} else {
		label = "very shallow"
	}

	if imgui.SliderFloatV("##bevelSize", &f, 0.005, 0.025, label, 1.0) {
		win.img.displayPrefs.CRT.BevelSize.Set(f)
	}
}

func (win *winPrefs) drawShine() {
	b := win.img.displayPrefs.CRT.Shine.Get().(bool)
	if imgui.Checkbox("Shine##shine", &b) {
		win.img.displayPrefs.CRT.Shine.Set(b)
	}
}

func (win *winPrefs) drawPixelPerfect() bool {
	b := !win.img.displayPrefs.CRT.Enabled.Get().(bool)
	if imgui.Checkbox("Pixel Perfect##pixelpefect", &b) {
		win.img.displayPrefs.CRT.Enabled.Set(!b)
	}

	if win.img.displayPrefs.CRT.Enabled.Get().(bool) {
		imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
		imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
		defer imgui.PopStyleVar()
		defer imgui.PopItemFlag()
	}

	imgui.SameLineV(0, 25)

	f := float32(win.img.displayPrefs.CRT.PixelPerfectFade.Get().(float64))

	var label string
	if f > 0.7 {
		label = "extreme fade"
	} else if f >= 0.4 {
		label = "high fade"
	} else if f > 0.0 {
		label = "tiny fade"
	} else if f == 0.0 {
		label = "no fade"
	}

	imgui.PushItemWidth(imguiRemainingWinWidth() * 0.75)
	if imgui.SliderFloatV("##pixelperfectfade", &f, 0.0, 0.9, label, 1.0) {
		win.img.displayPrefs.CRT.PixelPerfectFade.Set(f)
	}
	imgui.PopItemWidth()

	win.img.imguiTooltipSimple(`The fade slider controls how quickly pixels fade to
black. It is similar to the phosphor option that is
available when 'Pixel Perfect' mode is disabled.`)

	return b
}
