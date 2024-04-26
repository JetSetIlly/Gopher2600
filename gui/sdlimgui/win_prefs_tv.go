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
	"fmt"
	"math"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui/fonts"
)

func (win *winPrefs) drawTV() {
	imgui.Spacing()

	win.drawBrightness()
	imgui.Spacing()
	win.drawContrast()
	imgui.Spacing()
	win.drawHue()
}

func (win *winPrefs) drawBrightness() {
	imgui.Text(fmt.Sprintf("%c Brightness", fonts.TVBrightness))

	f := float32(win.img.crtPrefs.Brightness.Get().(float64))

	minv := float32(0.1)
	maxv := float32(1.90)
	label := fmt.Sprintf("%.0f", 100*(f-minv)/(maxv-minv))

	if imgui.SliderFloatV("##brightness", &f, minv, maxv, label, imgui.SliderFlagsNone) {
		win.img.crtPrefs.Brightness.Set(f)
	}
}

func (win *winPrefs) drawContrast() {
	imgui.Text(fmt.Sprintf("%c Contrast", fonts.TVContrast))

	f := float32(win.img.crtPrefs.Contrast.Get().(float64))

	minv := float32(0.1)
	maxv := float32(1.90)
	label := fmt.Sprintf("%.0f", 100*(f-minv)/(maxv-minv))

	if imgui.SliderFloatV("##contrast", &f, minv, maxv, label, imgui.SliderFlagsNone) {
		win.img.crtPrefs.Contrast.Set(f)
	}
}

func (win *winPrefs) drawHue() {
	imgui.Text(fmt.Sprintf("%c Hue", fonts.TVHue))

	f := float32(win.img.crtPrefs.Hue.Get().(float64))

	minv := float32(-0.99)
	maxv := float32(0.99)
	aminv := float32(math.Abs(float64(minv)))
	amaxv := float32(math.Abs(float64(maxv)))
	label := fmt.Sprintf("%.0f\u00b0", (f+minv+maxv)/(aminv+amaxv)*360)

	if imgui.SliderFloatV("##hue", &f, minv, maxv, label, imgui.SliderFlagsNone) {
		win.img.crtPrefs.Hue.Set(f)
	}
}
