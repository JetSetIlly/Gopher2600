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

	"github.com/inkyblackness/imgui-go/v2"
	"github.com/jetsetilly/gopher2600/logger"
)

const winCRTPrefsTitle = "CRT Preferences"

type winCRTPrefs struct {
	windowManagement
	img *SdlImgui
}

func newWinCRTPrefs(img *SdlImgui) (managedWindow, error) {
	win := &winCRTPrefs{
		img: img,
	}

	return win, nil
}

func (win *winCRTPrefs) init() {
}

func (win *winCRTPrefs) destroy() {
}

func (win *winCRTPrefs) id() string {
	return winCRTPrefsTitle
}

func (win *winCRTPrefs) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{10, 10}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winCRTPrefsTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	win.drawGamma()

	imgui.Spacing()
	imgui.Spacing()

	win.drawMask()

	imgui.Spacing()
	imgui.Spacing()

	win.drawScanlines()

	imgui.Spacing()
	imgui.Spacing()

	win.drawNoise()

	imgui.Spacing()
	imgui.Spacing()

	b := win.img.crtPrefs.Vignette.Get().(bool)
	if imgui.Checkbox("Vignette##vignette", &b) {
		win.img.crtPrefs.Vignette.Set(b)
	}

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	win.drawDiskButtons()

	imgui.End()
}

func (win *winCRTPrefs) drawGamma() {
	f := float32(win.img.crtPrefs.InputGamma.Get().(float64))
	if imgui.SliderFloatV("Input Gamma##input Gamma", &f, 1.0, 3.0, "%.2f", 1.0) {
		win.img.crtPrefs.InputGamma.Set(f)
	}

	f = float32(win.img.crtPrefs.OutputGamma.Get().(float64))
	if imgui.SliderFloatV("Output Gamma##output Gamma", &f, 1.0, 3.0, "%.2f", 1.0) {
		win.img.crtPrefs.OutputGamma.Set(f)
	}
}

func (win *winCRTPrefs) drawMask() {
	b := win.img.crtPrefs.Mask.Get().(bool)
	if imgui.Checkbox("Shadow Mask##mask", &b) {
		win.img.crtPrefs.Mask.Set(b)
	}
	f := float32(win.img.crtPrefs.MaskBrightness.Get().(float64))
	if imgui.SliderFloatV("Brightness##maskbrightness", &f, 0.0, 1.0, "%.2f", 1.0) {
		win.img.crtPrefs.MaskBrightness.Set(f)
	}
}

func (win *winCRTPrefs) drawScanlines() {
	b := win.img.crtPrefs.Scanlines.Get().(bool)
	if imgui.Checkbox("Scanlines##scanlines", &b) {
		win.img.crtPrefs.Scanlines.Set(b)
	}
	f := float32(win.img.crtPrefs.ScanlinesBrightness.Get().(float64))
	if imgui.SliderFloatV("Brightness##scanlinesbrightness", &f, 0.0, 1.0, "%.2f", 1.0) {
		win.img.crtPrefs.ScanlinesBrightness.Set(f)
	}
}

func (win *winCRTPrefs) drawNoise() {
	b := win.img.crtPrefs.Noise.Get().(bool)
	if imgui.Checkbox("Noise##noise", &b) {
		win.img.crtPrefs.Noise.Set(b)
	}
	f := float32(win.img.crtPrefs.NoiseLevel.Get().(float64))
	if imgui.SliderFloatV("Level##noiselevel", &f, 0.0, 1.0, "%.2f", 1.0) {
		win.img.crtPrefs.NoiseLevel.Set(f)
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
