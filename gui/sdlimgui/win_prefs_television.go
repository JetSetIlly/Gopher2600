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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
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
	imgui.Spacing()
	win.drawHaltConditions()
}

func (win *winPrefs) drawColour() {
	win.drawBrightness()
	imgui.Spacing()
	win.drawContrast()
	imgui.Spacing()
	win.drawSaturation()
	imgui.Spacing()
	win.drawHue()

	switch win.img.cache.TV.GetReqSpecID() {
	case specification.SpecNTSC.ID:
		imgui.Spacing()
		if imgui.CollapsingHeader("NTSC Colour Signal") {
			imgui.Spacing()
			win.drawNTSCPhase()
		}
	case specification.SpecPAL.ID:
		imgui.Spacing()
		if imgui.CollapsingHeader("PAL Colour Signal") {
			imgui.Spacing()
			win.drawPALPhase()
		}
	}
}

func (win *winPrefs) drawBrightness() {
	imgui.BeginGroup()
	defer imgui.EndGroup()

	imgui.Text(fmt.Sprintf("%c Brightness", fonts.TVBrightness))

	f := float32(win.img.displayPrefs.Colour.Brightness.Get().(float64))

	minv := float32(0.1)
	maxv := float32(1.90)
	label := fmt.Sprintf("%.0f", 100*(f-minv)/(maxv-minv))

	if imgui.SliderFloatV("##brightness", &f, minv, maxv, label, imgui.SliderFlagsNone) {
		win.img.displayPrefs.Colour.Brightness.Set(f)
	}
}

func (win *winPrefs) drawContrast() {
	imgui.BeginGroup()
	defer imgui.EndGroup()

	imgui.Text(fmt.Sprintf("%c Contrast", fonts.TVContrast))

	f := float32(win.img.displayPrefs.Colour.Contrast.Get().(float64))

	minv := float32(0.1)
	maxv := float32(1.90)
	label := fmt.Sprintf("%.0f", 100*(f-minv)/(maxv-minv))

	if imgui.SliderFloatV("##contrast", &f, minv, maxv, label, imgui.SliderFlagsNone) {
		win.img.displayPrefs.Colour.Contrast.Set(f)
	}
}

func (win *winPrefs) drawSaturation() {
	imgui.BeginGroup()
	defer imgui.EndGroup()

	imgui.Text(fmt.Sprintf("%c Saturation", fonts.TVSaturation))

	f := float32(win.img.displayPrefs.Colour.Saturation.Get().(float64))

	minv := float32(0.1)
	maxv := float32(1.90)
	label := fmt.Sprintf("%.0f", 100*(f-minv)/(maxv-minv))

	if imgui.SliderFloatV("##saturation", &f, minv, maxv, label, imgui.SliderFlagsNone) {
		win.img.displayPrefs.Colour.Saturation.Set(f)
	}
}

func (win *winPrefs) drawHue() {
	imgui.BeginGroup()
	defer imgui.EndGroup()

	imgui.Text(fmt.Sprintf("%c Hue", fonts.TVHue))

	f := float32(win.img.displayPrefs.Colour.Hue.Get().(float64))

	minv := float32(-0.99)
	maxv := float32(0.99)
	aminv := float32(math.Abs(float64(minv)))
	amaxv := float32(math.Abs(float64(maxv)))
	label := fmt.Sprintf("%.0f\u00b0", (f+minv+maxv)/(aminv+amaxv)*360)

	if imgui.SliderFloatV("##hue", &f, minv, maxv, label, imgui.SliderFlagsNone) {
		win.img.displayPrefs.Colour.Hue.Set(f)
	}
}

func (win *winPrefs) drawNTSCPhase() {
	imgui.BeginGroup()
	defer imgui.EndGroup()

	f := float32(specification.NTSCPhase)

	imgui.AlignTextToFramePadding()
	imgui.Text("Phase")
	imgui.SameLineV(0, 5)

	label := fmt.Sprintf("%.1f\u00b0", f)
	if imgui.SliderFloatV("##ntsc_phase", &f, 20.0, 30.0, label, imgui.SliderFlagsNone) {
		specification.NTSCPhase = float64(f)
	}

	imgui.Spacing()
	switch f {
	case specification.NTSCFieldService:
		label = specification.NTSCFieldSericeLabel
	case specification.NTSCVideoSoft:
		label = specification.NTSCVidoSoftLabel
	case specification.NTSCIdealDistribution:
		label = specification.NTSCIdealDistributionLabel
	default:
		label = "Custom"
	}

	imgui.AlignTextToFramePadding()
	imgui.Text("Preset")
	imgui.SameLineV(0, 5)

	if imgui.BeginComboV("##ntscpreset", label, imgui.ComboFlagsNone) {
		if imgui.Selectable(specification.NTSCFieldSericeLabel) {
			specification.NTSCPhase = specification.NTSCFieldService
		}
		if imgui.Selectable(specification.NTSCVidoSoftLabel) {
			specification.NTSCPhase = specification.NTSCVideoSoft
		}
		if imgui.Selectable(specification.NTSCIdealDistributionLabel) {
			specification.NTSCPhase = specification.NTSCIdealDistribution
		}
		imgui.EndCombo()
	}
}

func (win *winPrefs) drawPALPhase() {
	imgui.BeginGroup()
	defer imgui.EndGroup()

	f := float32(specification.PALPhase)

	imgui.AlignTextToFramePadding()
	imgui.Text("Phase")
	imgui.SameLineV(0, 5)

	label := fmt.Sprintf("%.1f\u00b0", f)
	if imgui.SliderFloatV("##pal_phase", &f, 10.0, 30.0, label, imgui.SliderFlagsNone) {
		specification.PALPhase = float64(f)
	}
}

func (win *winPrefs) drawHaltConditions() {
	if imgui.CollapsingHeader("Halting") {
		imgui.Spacing()
		prefsCheckbox(&win.img.dbg.VCS().Env.Prefs.TV.HaltVSYNCTooShort, "VSYNC too short")
		prefsCheckbox(&win.img.dbg.VCS().Env.Prefs.TV.HaltVSYNCScanlineStart, "VSYNC start scanline changes")
		prefsCheckbox(&win.img.dbg.VCS().Env.Prefs.TV.HaltVSYNCScanlineCount, "VSYNC scanline count changes")
		prefsCheckbox(&win.img.dbg.VCS().Env.Prefs.TV.HaltVSYNCabsent, "VSYNC absent")
		prefsCheckbox(&win.img.dbg.VCS().Env.Prefs.TV.HaltChangedVBLANK, "VBLANK bounds change")
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

		win.img.imguiTooltipSimple(`The number of scanlines for which VSYNC must be enabled
for it to be a valid VSYNC signal`)

		imgui.Spacing()
		imgui.Text("Speed of Recovery")
		recovery := int32(win.img.dbg.VCS().Env.Prefs.TV.VSYNCrecovery.Get().(int))

		const (
			verySlow  = 90
			slow      = 75
			quick     = 60
			veryQuick = 45
		)

		if recovery >= verySlow {
			recovery = 3
			label = fmt.Sprintf("very slow")
		} else if recovery >= slow {
			recovery = 2
			label = fmt.Sprintf("slow")
		} else if recovery >= quick {
			recovery = 1
			label = fmt.Sprintf("quick")
		} else if recovery >= veryQuick {
			recovery = 0
			label = fmt.Sprintf("very quick")
		}

		if imgui.SliderIntV("##vsyncRecover", &recovery, 0, 3, label, 1.0) {
			if recovery >= 3 {
				recovery = verySlow
			} else if recovery == 2 {
				recovery = slow
			} else if recovery == 1 {
				recovery = quick
			} else if recovery == 0 {
				recovery = veryQuick
			}
			win.img.dbg.VCS().Env.Prefs.TV.VSYNCrecovery.Set(recovery)
		}

		win.img.imguiTooltipSimple(`The speed at which the TV synchronises after
receiving a valid VSYNC signal`)

		imgui.Spacing()

		prefsCheckbox(&win.img.dbg.VCS().Env.Prefs.TV.VSYNCsyncedOnStart, "Synchronised on start")
		win.img.imguiTooltipSimple(`The television is synchronised on start`)
	}
}
