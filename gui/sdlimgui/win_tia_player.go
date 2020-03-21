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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlimgui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/tia/video"

	"github.com/inkyblackness/imgui-go/v2"
)

func (win *winTIA) drawPlayer(num int) {
	// get drawlist for window. we use this to draw index pointer
	// and horizpos indicator
	dl := imgui.WindowDrawList()

	lz := win.img.lazy.Player0
	ps := win.img.lazy.VCS.TIA.Video.Player0
	ms := win.img.lazy.VCS.TIA.Video.Missile0
	if num != 0 {
		lz = win.img.lazy.Player1
		ps = win.img.lazy.VCS.TIA.Video.Player1
		ms = win.img.lazy.VCS.TIA.Video.Missile1
	}

	imgui.Spacing()

	// player color indicator. when clicked popup palette is requested. on
	// selection in palette, missile color is changed too
	imguiText("Colour")
	col := lz.Color
	if win.img.imguiSwatch(col) {
		win.popupPalette.request(&col, func() {
			win.img.lazy.Dbg.PushRawEvent(func() { ps.Color = col })

			// update missile color too
			win.img.lazy.Dbg.PushRawEvent(func() { ms.Color = col })
		})
	}

	imguiText("Reflected")
	ref := lz.Reflected
	if imgui.Checkbox("##reflected", &ref) {
		win.img.lazy.Dbg.PushRawEvent(func() { ps.Reflected = ref })
	}

	imgui.SameLine()
	imguiText("Vert. Delay")
	vd := lz.VerticalDelay
	if imgui.Checkbox("##vertdelay", &vd) {
		// vertical delay affects which gfx register to use. set vertical delay
		// using the SetVerticalDelay function
		win.img.lazy.Dbg.PushRawEvent(func() { ps.SetVerticalDelay(vd) })
	}

	imgui.Spacing()
	imgui.Spacing()

	// hmove value
	imguiText("HMOVE")
	imgui.SameLine()
	imgui.PushItemWidth(win.byteDim.X)
	hmove := fmt.Sprintf("%01x", lz.Hmove)
	if imguiHexInput("##hmove", !win.img.paused, 1, &hmove) {
		if v, err := strconv.ParseUint(hmove, 16, 8); err == nil {
			win.img.lazy.Dbg.PushRawEvent(func() { ps.Hmove = uint8(v) })
		}
	}
	imgui.PopItemWidth()

	// hmove slider
	imgui.SameLine()
	imgui.PushItemWidth(win.hmoveSliderWidth)
	hmoveSlider := int32(lz.Hmove) - 8
	if imgui.SliderIntV("##hmoveslider", &hmoveSlider, -8, 7, "%d") {
		win.img.lazy.Dbg.PushRawEvent(func() { ps.Hmove = uint8(hmoveSlider + 8) })
	}
	imgui.PopItemWidth()

	imgui.Spacing()
	imgui.Spacing()

	// graphics data - new
	imguiText("New Gfx")
	ngfxSeq := newDrawlistSequence(win.img, imgui.Vec2{X: imgui.FrameHeight(), Y: imgui.FrameHeight()}, 0.1)
	od := lz.GfxDataNew
	for i := 0; i < 8; i++ {
		var col uint8
		if (od<<i)&0x80 == 0x80 {
			col = lz.Color
		} else {
			col = 0x0
			ngfxSeq.nextItemDepressed = true
		}
		if ngfxSeq.rectFilled(col) {
			od ^= 0x80 >> i
			win.img.lazy.Dbg.PushRawEvent(func() { ps.GfxDataNew = od })
		}
		ngfxSeq.sameLine()

		// deliberately not using setGfxData() function from Player type. it
		// woulnd't make sense in this debugging context to do that.
	}
	ngfxSeq.end()

	// graphics data - old
	imgui.SameLine()
	imguiText("Old Gfx")
	ogfxSeq := newDrawlistSequence(win.img, imgui.Vec2{X: imgui.FrameHeight(), Y: imgui.FrameHeight()}, 0.1)
	nd := lz.GfxDataOld
	for i := 0; i < 8; i++ {
		var col uint8
		if (nd<<i)&0x80 == 0x80 {
			col = lz.Color
		} else {
			col = 0x0
			ogfxSeq.nextItemDepressed = true
		}
		if ogfxSeq.rectFilled(col) {
			nd ^= 0x80 >> i
			win.img.lazy.Dbg.PushRawEvent(func() { ps.GfxDataOld = nd })
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
			idx = 7 - lz.ScanPixel
		} else {
			idx = lz.ScanPixel
		}

		seq := ngfxSeq
		if lz.VerticalDelay {
			seq = ogfxSeq
		}
		pt := imgui.Vec2{
			X: seq.offsetX(idx),
			Y: imgui.CursorScreenPos().Y,
		}

		dl.AddCircleFilled(pt, imgui.FontSize()*0.20, win.idxPointer)
	}

	imgui.Spacing()
	imgui.Spacing()

	// nusiz
	imgui.BeginGroup()
	imgui.PushItemWidth(win.playerSizeAndCopiesComboDim.X)
	if imgui.BeginComboV("##playersizecopies", video.PlayerSizes[lz.SizeAndCopies], imgui.ComboFlagNoArrowButton) {
		for k := range video.PlayerSizes {
			if imgui.Selectable(video.PlayerSizes[k]) {
				v := uint8(k) // being careful about scope
				win.img.lazy.Dbg.PushRawEvent(func() {
					ps.SizeAndCopies = v
					win.img.lazy.VCS.TIA.Video.UpdateNUSIZ(num, false)
				})
			}
		}

		imgui.EndCombo()
	}
	imgui.PopItemWidth()

	imgui.SameLine()
	imguiText("NUSIZ")
	imgui.SameLine()
	imgui.PushItemWidth(win.byteDim.X)
	nusiz := fmt.Sprintf("%02x", lz.Nusiz)
	if imguiHexInput("##nusiz", !win.img.paused, 2, &nusiz) {
		if v, err := strconv.ParseUint(nusiz, 16, 8); err == nil {
			win.img.lazy.Dbg.PushRawEvent(func() {
				ps.SetNUSIZ(uint8(v))

				// update missile NUSIZ too
				ms.SetNUSIZ(uint8(v))
			})
		}
	}
	imgui.PopItemWidth()
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
	imgui.Text(fmt.Sprintf("Last reset at pixel %03d. Draws at pixel %03d", lz.ResetPixel, lz.HmovedPixel))

	if lz.MoreHmove {
		imgui.SameLine()
		imgui.Text("[currently moving]")
	}
}
