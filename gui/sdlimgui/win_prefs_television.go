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
	"math"

	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/television/colourgen"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/imgui-go/v5"
)

func (win *winPrefs) drawTelevision() {
	pos := imgui.CursorScreenPos()
	pos.X += imgui.WindowWidth()
	defer func() {
		imgui.SetNextWindowPos(pos)
		if imgui.BeginV("##prefspalette", &win.playmodeOpen, imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoDecoration) {
			p := newPalette(win.img)
			p.draw(paletteNoSelection)
		}
		imgui.End()
	}()

	imgui.PushItemWidth(400)
	defer imgui.PopItemWidth()

	imgui.Spacing()
	win.drawColour()
	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()
	win.drawVSYNC()
}

func (win *winPrefs) drawColour() {
	// select which adjustments settings to use
	adjust := &specification.ColourGen.LegacyAdjust

	win.drawBrightness(adjust)
	imgui.Spacing()
	win.drawContrast(adjust)
	imgui.Spacing()
	win.drawSaturation(adjust)
	imgui.Spacing()
	win.drawHue(adjust)

	switch win.img.cache.TV.GetFrameInfo().Spec.ID {
	case specification.SpecNTSC.ID:
		imgui.Spacing()
		if imgui.CollapsingHeader("NTSC Colour Signal") {
			imgui.Spacing()
			win.drawNTSCPhaseAdj()
		}
	case specification.SpecPAL.ID:
		imgui.Spacing()
		if imgui.CollapsingHeader("PAL Colour Signal") {
			imgui.Spacing()
			win.drawPALPhaseAdj()
		}
	}
}

func (win *winPrefs) drawBrightness(adjust *colourgen.Adjust) {
	imgui.BeginGroup()
	defer imgui.EndGroup()

	imgui.Text(fmt.Sprintf("%c Brightness", fonts.TVBrightness))

	f := float32(adjust.Brightness.Get().(float64))

	minv := float32(0.1)
	maxv := float32(1.9)
	label := fmt.Sprintf("%.0f", 100*(f-minv)/(maxv-minv))

	if imgui.SliderFloatV("##brightness", &f, minv, maxv, label, imgui.SliderFlagsNone) {
		adjust.Brightness.Set(f)
	}
}

func (win *winPrefs) drawContrast(adjust *colourgen.Adjust) {
	imgui.BeginGroup()
	defer imgui.EndGroup()

	imgui.Text(fmt.Sprintf("%c Contrast", fonts.TVContrast))

	f := float32(adjust.Contrast.Get().(float64))

	minv := float32(0.1)
	maxv := float32(1.90)
	label := fmt.Sprintf("%.0f", 100*(f-minv)/(maxv-minv))

	if imgui.SliderFloatV("##contrast", &f, minv, maxv, label, imgui.SliderFlagsNone) {
		adjust.Contrast.Set(f)
	}
}

func (win *winPrefs) drawSaturation(adjust *colourgen.Adjust) {
	imgui.BeginGroup()
	defer imgui.EndGroup()

	imgui.Text(fmt.Sprintf("%c Saturation", fonts.TVSaturation))

	f := float32(adjust.Saturation.Get().(float64))

	minv := float32(0.1)
	maxv := float32(1.90)
	label := fmt.Sprintf("%.0f", 100*(f-minv)/(maxv-minv))

	if imgui.SliderFloatV("##saturation", &f, minv, maxv, label, imgui.SliderFlagsNone) {
		adjust.Saturation.Set(f)
	}
}

func (win *winPrefs) drawHue(adjust *colourgen.Adjust) {
	imgui.BeginGroup()
	defer imgui.EndGroup()

	imgui.Text(fmt.Sprintf("%c Hue", fonts.TVHue))

	f := float32(adjust.Hue.Get().(float64))

	minv := float32(-180)
	maxv := float32(180)
	aminv := float32(math.Abs(float64(minv)))
	amaxv := float32(math.Abs(float64(maxv)))
	label := fmt.Sprintf("%.0f\u00b0", (f+minv+maxv)/(aminv+amaxv)*360)

	if imgui.SliderFloatV("##hue", &f, minv, maxv, label, imgui.SliderFlagsNone) {
		adjust.Hue.Set(f)
	}
}

func (win *winPrefs) drawNTSCPhaseAdj() {
	imgui.BeginGroup()
	defer imgui.EndGroup()

	f := float32(specification.ColourGen.LegacyAdjust.NTSCPhase.Get().(float64))

	imgui.AlignTextToFramePadding()
	imgui.Text("Phase Adjust")
	imgui.SameLineV(0, 5)

	if imgui.SliderFloatV("##ntscphaseadj", &f, -10, 10, "%0.2f", imgui.SliderFlagsNone) {
		specification.ColourGen.LegacyAdjust.NTSCPhase.Set(f)
	}
}

func (win *winPrefs) drawPALPhaseAdj() {
	imgui.BeginGroup()
	defer imgui.EndGroup()

	f := float32(specification.ColourGen.LegacyAdjust.PALPhase.Get().(float64))

	imgui.AlignTextToFramePadding()
	imgui.Text("Phase Adjust")
	imgui.SameLineV(0, 5)

	if imgui.SliderFloatV("##palphaseadj", &f, -10, 10, "%0.2f", imgui.SliderFlagsNone) {
		specification.ColourGen.LegacyAdjust.PALPhase.Set(f)
	}
}

func (win *winPrefs) drawVSYNC() {
	var label string

	if imgui.CollapsingHeader("Synchronisation") {
		imgui.Spacing()
		imgui.Text("VSYNC Scanlines Required")
		scanlines := int32(win.img.dbg.VCS().Env.Prefs.TV.VSYNCscanlines.Get().(int))

		if scanlines == 1 {
			label = fmt.Sprintf("%d scanline", scanlines)
		} else {
			label = fmt.Sprintf("%d scanlines", scanlines)
		}

		if imgui.SliderIntV("##vsyncScanlines", &scanlines, 0, 4, label, 1.0) {
			win.img.dbg.VCS().Env.Prefs.TV.VSYNCscanlines.Set(scanlines)
		}
		win.img.imguiTooltipSimple("Number of scanlines for valid VSYNC")

		imgui.Spacing()
		prefsCheckbox(&win.img.dbg.VCS().Env.Prefs.TV.VSYNCimmedateSync, "Immediate Synchronisation")
		win.img.imguiTooltipSimple("Whether the screen should synchroise immediately")

		drawDisabled(win.img.dbg.VCS().Env.Prefs.TV.VSYNCimmedateSync.Get().(bool), func() {
			prefsCheckbox(&win.img.dbg.VCS().Env.Prefs.TV.VSYNCsyncedOnStart, "Synchronised on start")
			win.img.imguiTooltipSimple("No visible synchronisation on start")
		})

		prefsCheckbox(&win.img.dbg.VCS().Env.Prefs.TV.HaltChangedVSYNC, "Abort on bad VSYNC")
		win.img.imguiTooltipSimple("Enter debugger when VSYNC is bad")

		prefsCheckbox(&win.img.dbg.VCS().Env.Prefs.TV.HaltChangedVBLANK, "Abort on changed VBLANK bounds")
		win.img.imguiTooltipSimple("Enter debugger when use of VBLANK changes")
	}
}
