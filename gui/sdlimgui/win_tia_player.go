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

	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/tia/video"

	"github.com/inkyblackness/imgui-go/v4"
)

func (win *winTIA) drawPlayer(num int) {
	imgui.BeginChildV(fmt.Sprintf("##player%d", num), imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.scopeHeight}, false, 0)
	defer imgui.EndChild()

	// get drawlist for window. we use this to draw index pointer and position indicator
	dl := imgui.WindowDrawList()

	lz := win.img.lz.Player0
	ps := win.img.lz.Player0.Ps
	ms := win.img.lz.Missile0.Ms
	if num != 0 {
		lz = win.img.lz.Player1
		ps = win.img.lz.Player1.Ps
		ms = win.img.lz.Missile1.Ms
	}

	imgui.Spacing()

	// player color indicator. when clicked popup palette is requested. on
	// selection in palette, missile color is changed too
	imguiLabel("Colour")
	col := lz.Color
	if win.img.imguiSwatch(col, 0.75) {
		win.popupPalette.request(&col, func() {
			win.img.dbg.PushRawEvent(func() { ps.Color = col })

			// update missile color too
			win.img.dbg.PushRawEvent(func() { ms.Color = col })
		})
	}

	imguiLabel("Reflected")
	ref := lz.Reflected
	if imgui.Checkbox("##reflected", &ref) {
		win.update(
			func() {
				var o uint8
				if ps.Reflected {
					o = video.REFPxMask
				}
				var n uint8
				if ref {
					n = video.REFPxMask
				}
				var reg cpubus.Register
				switch num {
				case 0:
					reg = cpubus.REFP0
				case 1:
					reg = cpubus.REFP1
				default:
					panic("unexecpted player number")
				}
				win.img.dbg.PushDeepPoke(cpubus.WriteAddress[reg], o, n, video.REFPxMask)
			},
			func() {
				ps.Reflected = ref
			})
	}

	imgui.SameLine()
	imguiLabel("Vert. Delay")
	vd := lz.VerticalDelay
	if imgui.Checkbox("##vertdelay", &vd) {
		// vertical delay affects which gfx register to use. set vertical delay
		// using the SetVerticalDelay function
		win.img.dbg.PushRawEvent(func() { ps.SetVerticalDelay(vd) })
	}

	imgui.Spacing()
	imgui.Spacing()

	// hmove value
	imguiLabel("HMOVE")
	imgui.SameLine()
	hmove := fmt.Sprintf("%01x", lz.Hmove)
	if imguiHexInput("##hmove", 1, &hmove) {
		if v, err := strconv.ParseUint(hmove, 16, 8); err == nil {
			win.img.dbg.PushRawEvent(func() { ps.Hmove = uint8(v) })
		}
	}

	// hmove slider
	imgui.SameLine()
	imgui.PushItemWidth(win.hmoveSliderWidth)
	hmoveSlider := int32(lz.Hmove) - 8
	if imgui.SliderIntV("##hmoveslider", &hmoveSlider, -8, 7, "%d", imgui.SliderFlagsNone) {
		win.img.dbg.PushRawEvent(func() { ps.Hmove = uint8(hmoveSlider + 8) })
	}
	imgui.PopItemWidth()

	imgui.Spacing()
	imgui.Spacing()

	// tv palette used to draw bit sequences with correct colours
	_, palette, _ := win.img.imguiTVPalette()

	// graphics data - new
	imguiLabel("New Gfx")
	ngfxSeq := newDrawlistSequence(imgui.Vec2{X: imgui.FrameHeight(), Y: imgui.FrameHeight()}, false)
	od := lz.GfxDataNew
	for i := 0; i < 8; i++ {
		var col uint8
		if (od<<i)&0x80 == 0x80 {
			col = lz.Color
		} else {
			col = 0x0
			ngfxSeq.nextItemDepressed = true
		}
		if ngfxSeq.rectFill(palette[col]) {
			od ^= 0x80 >> i
			win.img.dbg.PushRawEvent(func() { ps.GfxDataNew = od })
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
	nd := lz.GfxDataOld
	for i := 0; i < 8; i++ {
		var col uint8
		if (nd<<i)&0x80 == 0x80 {
			col = lz.Color
		} else {
			col = 0x0
			ogfxSeq.nextItemDepressed = true
		}
		if ogfxSeq.rectFill(palette[col]) {
			nd ^= 0x80 >> i
			win.img.dbg.PushRawEvent(func() { ps.GfxDataOld = nd })
		}
		ogfxSeq.sameLine()

		// deliberately not using setGfxData() function from Player type. it
		// woulnd't make sense in this debugging context to do that.
	}
	ogfxSeq.end()

	// scancounter index pointer
	if lz.ScanIsActive {
		var idx int
		if lz.Reflected {
			idx = lz.ScanPixel
		} else {
			idx = 7 - lz.ScanPixel
		}

		seq := ngfxSeq
		if lz.VerticalDelay {
			seq = ogfxSeq
		}
		pt := imgui.Vec2{
			X: seq.offsetX(idx),
			Y: imgui.CursorScreenPos().Y,
		}

		dl.AddCircleFilled(pt, imgui.FontSize()*0.20, win.img.cols.tiaPointer)
	}

	imgui.Spacing()
	imgui.Spacing()

	// nusiz
	imgui.BeginGroup()
	imgui.PushItemWidth(win.playerSizeAndCopiesComboDim.X)
	if imgui.BeginComboV("##playersizecopies", video.PlayerSizes[lz.SizeAndCopies], imgui.ComboFlagsNoArrowButton) {
		for k := range video.PlayerSizes {
			if imgui.Selectable(video.PlayerSizes[k]) {
				v := uint8(k) // being careful about scope
				win.img.dbg.PushRawEvent(func() {
					ps.SizeAndCopies = v
					win.img.vcs.TIA.Video.UpdateNUSIZ(num, false)
				})
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()

	imgui.SameLine()
	imguiLabel("NUSIZ")
	imgui.SameLine()
	nusiz := fmt.Sprintf("%02x", lz.Nusiz)
	if imguiHexInput("##nusiz", 2, &nusiz) {
		if v, err := strconv.ParseUint(nusiz, 16, 8); err == nil {
			win.img.dbg.PushRawEvent(func() {
				ps.SetNUSIZ(uint8(v))

				// update missile NUSIZ too
				ms.SetNUSIZ(uint8(v))
			})
		}
	}
	imgui.SameLine()

	s := strings.Builder{}
	if lz.ScanIsActive || lz.ScanIsLatching {
		if lz.ScanIsActive {
			s.WriteString("drawing ")
			if lz.Nusiz != lz.ScanLatchedSizeAndCopies {
				// add a note to indicate that the nusiz value is about to update
				s.WriteString("[prev. nusiz] ")
			}
		} else {
			s.WriteString("latching ")
		}

		switch lz.ScanCpy {
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
	imgui.Text(fmt.Sprintf("Last reset at clock %03d. First copy draws at clock %03d", lz.ResetPixel, lz.HmovedPixel))

	if lz.MoreHmove {
		imgui.SameLine()
		imgui.Text("[currently moving]")
	}
}
