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
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/tia/video"

	"github.com/inkyblackness/imgui-go/v4"
)

func (win *winTIA) drawBall() {
	lz := win.img.lz.Ball
	bs := win.img.lz.Ball.Bs
	pf := win.img.lz.Playfield.Pf

	imgui.Spacing()

	imgui.BeginGroup()
	imguiLabel("Colour")
	col := lz.Color
	if win.img.imguiSwatch(col, 0.75) {
		win.popupPalette.request(&col, func() {
			win.img.lz.Dbg.PushRawEvent(func() { bs.Color = col })

			// update playfield color too
			win.img.lz.Dbg.PushRawEvent(func() { pf.ForegroundColor = col })
		})
	}

	imguiLabel("Enabled")
	enb := lz.Enabled
	if imgui.Checkbox("##enabled", &enb) {
		win.img.lz.Dbg.PushRawEvent(func() { bs.Enabled = enb })
	}

	imgui.SameLine()
	imguiLabel("Vert Del.")
	enbv := lz.VerticalDelay
	if imgui.Checkbox("##vdelay", &enbv) {
		win.img.lz.Dbg.PushRawEvent(func() { bs.VerticalDelay = enbv })
	}

	imgui.SameLine()
	imguiLabel("Enabled Del.")
	enbd := lz.EnabledDelay
	if imgui.Checkbox("##enableddelay", &enbd) {
		win.img.lz.Dbg.PushRawEvent(func() { bs.EnabledDelay = enbd })
	}
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// hmove value and slider
	imgui.BeginGroup()
	imguiLabel("HMOVE")
	imgui.SameLine()
	hmove := fmt.Sprintf("%01x", lz.Hmove)
	if imguiHexInput("##hmove", 1, &hmove) {
		if v, err := strconv.ParseUint(hmove, 16, 8); err == nil {
			win.img.lz.Dbg.PushRawEvent(func() { bs.Hmove = uint8(v) })
		}
	}

	imgui.SameLine()
	imgui.PushItemWidth(win.hmoveSliderWidth)
	hmoveSlider := int32(lz.Hmove) - 8
	if imgui.SliderIntV("##hmoveslider", &hmoveSlider, -8, 7, "%d", imgui.SliderFlagsNone) {
		win.img.lz.Dbg.PushRawEvent(func() { bs.Hmove = uint8(hmoveSlider + 8) })
	}
	imgui.PopItemWidth()
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// ctrlpf, size selector and drawing info
	imgui.BeginGroup()
	imgui.PushItemWidth(win.ballSizeComboDim.X)
	if imgui.BeginComboV("##ballsize", video.BallSizes[lz.Size], imgui.ComboFlagsNoArrowButton) {
		for k := range video.BallSizes {
			if imgui.Selectable(video.BallSizes[k]) {
				v := uint8(k) // being careful about scope
				win.img.lz.Dbg.PushRawEvent(func() {
					bs.Size = v
					win.img.lz.Dbg.VCS.TIA.Video.UpdateCTRLPF()
				})
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()

	imgui.SameLine()
	imguiLabel("CTRLPF")
	imgui.SameLine()
	ctrlpf := fmt.Sprintf("%02x", lz.Ctrlpf)
	if imguiHexInput("##ctrlpf", 2, &ctrlpf) {
		if v, err := strconv.ParseUint(ctrlpf, 16, 8); err == nil {
			win.img.lz.Dbg.PushRawEvent(func() {
				bs.SetCTRLPF(uint8(v))

				// update playfield CTRLPF too
				pf.SetCTRLPF(uint8(v))
			})
		}
	}

	s := strings.Builder{}
	if lz.EncActive {
		s.WriteString("drawing ")
		if lz.EncSecondHalf {
			s.WriteString("(2nd half)")
		}
		s.WriteString(fmt.Sprintf(" [%d]", lz.EncTicks))
	}
	imgui.SameLine()
	imgui.Text(s.String())
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// horizontal positioning
	imgui.BeginGroup()
	imgui.Text(fmt.Sprintf("Last reset at clock %03d. Draws at clock %03d", lz.ResetPixel, lz.HmovedPixel))
	if lz.MoreHmove {
		imgui.SameLine()
		imgui.Text("[currently moving]")
	}
	imgui.EndGroup()
}
