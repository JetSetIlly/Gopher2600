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

func (win *winTIA) drawPlayer(num int) {
	if num != 0 && num != 1 {
		panic(fmt.Sprintf("impossible player number %d", num))
	}

	var player *video.PlayerSprite

	switch num {
	case 0:
		player = win.img.cache.VCS.TIA.Video.Player0
	case 1:
		player = win.img.cache.VCS.TIA.Video.Player1
	}

	realPlayer := func() *video.PlayerSprite {
		switch num {
		case 0:
			return win.img.dbg.VCS().TIA.Video.Player0
		case 1:
			return win.img.dbg.VCS().TIA.Video.Player1
		}
		return nil
	}

	realMissile := func() *video.MissileSprite {
		switch num {
		case 0:
			return win.img.dbg.VCS().TIA.Video.Missile0
		case 1:
			return win.img.dbg.VCS().TIA.Video.Missile1
		}
		return nil
	}

	imgui.Spacing()

	// player color indicator. when clicked popup palette is requested. on
	// selection in palette, missile color is changed too
	imguiLabel("Colour")
	col := player.Color
	if win.img.imguiTVColourSwatch(col, 0.75) {
		win.popupPalette.request(&col, func() {
			win.img.dbg.PushFunction(func() {
				realPlayer().Color = col
			})
			win.img.dbg.PushFunction(func() {
				realMissile().Color = col
			})
		})
	}

	imguiLabel("Reflected")
	ref := player.Reflected
	if imgui.Checkbox("##reflected", &ref) {
		win.img.dbg.PushFunction(func() {
			realPlayer().Reflected = ref
		})
	}

	imgui.SameLine()
	imguiLabel("Vert. Delay")
	vd := player.VerticalDelay
	if imgui.Checkbox("##vertdelay", &vd) {
		// vertical delay affects which gfx register to use. set vertical delay
		// using the SetVerticalDelay function
		win.img.dbg.PushFunction(func() {
			realPlayer().SetVerticalDelay(vd)
		})
	}

	imgui.Spacing()
	imgui.Spacing()

	// hmove value
	imguiLabel("HMOVE")
	imgui.SameLine()
	hmove := fmt.Sprintf("%01x", player.Hmove)
	if imguiHexInput("##hmove", 1, &hmove) {
		if v, err := strconv.ParseUint(hmove, 16, 8); err == nil {
			win.img.dbg.PushFunction(func() {
				realPlayer().Hmove = uint8(v)
			})
		}
	}

	// hmove slider
	imgui.SameLine()
	imgui.PushItemWidth(win.hmoveSliderWidth)
	hmoveSlider := int32(player.Hmove) - 8
	if imgui.SliderIntV("##hmoveslider", &hmoveSlider, -8, 7, "%d", imgui.SliderFlagsNone) {
		win.img.dbg.PushFunction(func() {
			realPlayer().Hmove = uint8(hmoveSlider + 8)
		})
	}
	imgui.PopItemWidth()

	imgui.Spacing()
	imgui.Spacing()

	// graphics data - new
	imguiLabel("New Gfx")
	ngfxSeq := newDrawlistSequence(imgui.Vec2{X: imgui.FrameHeight(), Y: imgui.FrameHeight()}, false)
	od := player.GfxDataNew
	for i := 0; i < 8; i++ {
		var col uint8
		if (od<<i)&0x80 == 0x80 {
			col = player.Color
		} else {
			col = 0x0
			ngfxSeq.nextItemDepressed = true
		}
		if ngfxSeq.rectFill(win.img.getTVColour(col)) {
			od ^= 0x80 >> i
			win.img.dbg.PushFunction(func() {
				realPlayer().GfxDataNew = od
			})
		}
		ngfxSeq.sameLine()

		// deliberately not using setGfxData() function from Player type. it
		// woulnd't make sense in this debugging context to do that.
	}
	ngfxSeq.end()

	// graphics data - old
	imgui.SameLine()
	imguiLabel("Old Gfx")
	ogfxSeq := newDrawlistSequence(imgui.Vec2{X: imgui.FrameHeight(), Y: imgui.FrameHeight()}, false)
	nd := player.GfxDataOld
	for i := 0; i < 8; i++ {
		var col uint8
		if (nd<<i)&0x80 == 0x80 {
			col = player.Color
		} else {
			col = 0x0
			ogfxSeq.nextItemDepressed = true
		}
		if ogfxSeq.rectFill(win.img.getTVColour(col)) {
			nd ^= 0x80 >> i
			win.img.dbg.PushFunction(func() {
				realPlayer().GfxDataOld = nd
			})
		}
		ogfxSeq.sameLine()

		// deliberately not using setGfxData() function from Player type. it
		// woulnd't make sense in this debugging context to do that.
	}
	ogfxSeq.end()

	// scancounter index pointer
	if player.ScanCounter.IsActive() {
		var idx int
		if player.Reflected {
			idx = player.ScanCounter.Pixel
		} else {
			idx = 7 - player.ScanCounter.Pixel
		}

		seq := ngfxSeq
		if player.VerticalDelay {
			seq = ogfxSeq
		}
		pt := imgui.Vec2{
			X: seq.offsetX(idx),
			Y: imgui.CursorScreenPos().Y,
		}

		imgui.WindowDrawList().AddCircleFilled(pt, imgui.FontSize()*0.20, win.img.cols.tiaPointer)
	}

	imgui.Spacing()
	imgui.Spacing()

	// nusiz
	imgui.BeginGroup()
	imgui.PushItemWidth(win.playerSizeAndCopiesComboDim.X)
	if imgui.BeginComboV("##playersizecopies", video.PlayerSizes[player.SizeAndCopies], imgui.ComboFlagsNoArrowButton) {
		for k := range video.PlayerSizes {
			if imgui.Selectable(video.PlayerSizes[k]) {
				v := uint8(k) // being careful about scope
				win.img.dbg.PushFunction(func() {
					realPlayer().SizeAndCopies = v
					win.img.dbg.VCS().TIA.Video.UpdateNUSIZ(num, false)
				})
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()

	imgui.SameLine()
	imguiLabel("NUSIZ")
	imgui.SameLine()
	nusiz := fmt.Sprintf("%02x", player.Nusiz)
	if imguiHexInput("##nusiz", 2, &nusiz) {
		if v, err := strconv.ParseUint(nusiz, 16, 8); err == nil {
			win.img.dbg.PushFunction(func() {
				realPlayer().SetNUSIZ(uint8(v))
				realMissile().SetNUSIZ(uint8(v))
			})
		}
	}
	imgui.SameLine()

	s := strings.Builder{}
	if player.ScanCounter.IsActive() || player.ScanCounter.IsLatching() {
		if player.ScanCounter.IsActive() {
			s.WriteString("drawing ")
			if player.Nusiz != player.ScanCounter.LatchedSizeAndCopies {
				// add a note to indicate that the nusiz value is about to update
				s.WriteString("[prev. nusiz] ")
			}
		} else {
			s.WriteString("latching ")
		}

		switch player.ScanCounter.Cpy {
		case 0:
		case 1:
			s.WriteString("2nd copy")
		case 2:
			s.WriteString("3rd copy")
		default:
			panic("illegal number of player copies")
		}
	}
	imgui.Text(s.String())
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// horizontal positioning
	imgui.Text(fmt.Sprintf("Last reset at clock %03d. First copy draws at clock %03d", player.ResetPixel, player.HmovedPixel))

	if player.MoreHMOVE {
		imgui.SameLine()
		imgui.Text("[currently moving]")
	}
}
