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

func (win *winTIA) drawMissile(num int) {
	imgui.BeginChildV(fmt.Sprintf("##num%d", num), imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.footerHeight}, false, 0)
	defer imgui.EndChild()

	var missile *video.MissileSprite
	var realMissile *video.MissileSprite
	var realPlayer *video.PlayerSprite

	switch num {
	case 0:
		missile = win.img.cache.VCS.TIA.Video.Missile0
		realMissile = win.img.dbg.VCS().TIA.Video.Missile0
		realPlayer = win.img.dbg.VCS().TIA.Video.Player0
	case 1:
		missile = win.img.cache.VCS.TIA.Video.Missile1
		realMissile = win.img.dbg.VCS().TIA.Video.Missile1
		realPlayer = win.img.dbg.VCS().TIA.Video.Player1
	default:
		panic(fmt.Sprintf("impossible missile number %d", num))
	}

	imgui.Spacing()

	imgui.BeginGroup()
	imguiLabel("Colour")
	col := missile.Color
	if win.img.imguiSwatch(col, 0.75) {
		win.popupPalette.request(&col, func() {
			win.img.dbg.PushFunction(func() {
				realMissile.Color = col
				realPlayer.Color = col
			})
		})
	}

	imguiLabel("Reset-to-Player")
	r2p := missile.ResetToPlayer
	if imgui.Checkbox("##resettoplayer", &r2p) {
		win.img.dbg.PushFunction(func() { realMissile.ResetToPlayer = r2p })
	}

	imgui.SameLine()
	imguiLabel("Enabled")
	enb := missile.Enabled
	if imgui.Checkbox("##enabled", &enb) {
		win.img.dbg.PushFunction(func() { realMissile.Enabled = enb })
	}
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// hmove value and slider
	imgui.BeginGroup()
	imguiLabel("HMOVE")
	imgui.SameLine()
	hmove := fmt.Sprintf("%01x", missile.Hmove)
	if imguiHexInput("##hmove", 1, &hmove) {
		if v, err := strconv.ParseUint(hmove, 16, 8); err == nil {
			win.img.dbg.PushFunction(func() { realMissile.Hmove = uint8(v) })
		}
	}

	imgui.SameLine()
	imgui.PushItemWidth(win.hmoveSliderWidth)
	hmoveSlider := int32(missile.Hmove) - 8
	if imgui.SliderIntV("##hmoveslider", &hmoveSlider, -8, 7, "%d", imgui.SliderFlagsNone) {
		win.img.dbg.PushFunction(func() { realMissile.Hmove = uint8(hmoveSlider + 8) })
	}
	imgui.PopItemWidth()
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// nusiz
	imgui.BeginGroup()
	imgui.PushItemWidth(win.missileCopiesComboDim.X)
	if imgui.BeginComboV("##missilecopies", video.MissileCopies[missile.Copies], imgui.ComboFlagsNoArrowButton) {
		for k := range video.MissileCopies {
			if imgui.Selectable(video.MissileCopies[k]) {
				v := uint8(k) // being careful about scope
				win.img.dbg.PushFunction(func() {
					realMissile.Copies = v
					win.img.dbg.VCS().TIA.Video.UpdateNUSIZ(num, true)
				})
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()

	imgui.SameLine()
	imgui.PushItemWidth(win.missileSizeComboDim.X)
	if imgui.BeginComboV("##missilesize", video.MissileSizes[missile.Size], imgui.ComboFlagsNoArrowButton) {
		for k := range video.MissileSizes {
			if imgui.Selectable(video.MissileSizes[k]) {
				v := uint8(k) // being careful about scope
				win.img.dbg.PushFunction(func() {
					realMissile.Size = v
					win.img.dbg.VCS().TIA.Video.UpdateNUSIZ(num, true)
				})
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()

	imgui.SameLine()
	imguiLabel("NUSIZ")
	nusiz := fmt.Sprintf("%02x", missile.Nusiz)
	if imguiHexInput("##nusiz", 2, &nusiz) {
		if v, err := strconv.ParseUint(nusiz, 16, 8); err == nil {
			win.img.dbg.PushFunction(func() {
				realMissile.SetNUSIZ(uint8(v))
				realPlayer.SetNUSIZ(uint8(v))
			})
		}
	}

	s := strings.Builder{}
	if missile.Enclockifier.Active {
		s.WriteString("drawing ")
		if missile.Enclockifier.SecondHalf {
			s.WriteString("2nd half of ")
		}
		switch missile.Enclockifier.Cpy {
		case 0:
			s.WriteString("1st copy")
		case 1:
			s.WriteString("2nd copy")
		case 2:
			s.WriteString("3rd copy")
		}
		s.WriteString(fmt.Sprintf(" [%d]", missile.Enclockifier.Ticks))
	}
	imgui.SameLine()
	imgui.Text(s.String())
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// horizontal positioning
	imgui.BeginGroup()
	imgui.Text(fmt.Sprintf("Last reset at clock %03d. First copy draws at clock %03d", missile.ResetPixel, missile.HmovedPixel))
	if missile.MoreHMOVE {
		imgui.SameLine()
		imgui.Text("[currently moving]")
	}
	imgui.EndGroup()
}
