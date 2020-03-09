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

	"github.com/inkyblackness/imgui-go/v2"
)

func (win *winTIA) drawPlayer(player int) {
	// get drawlist for window. we use this to draw index pointer
	// and horizpos indicator
	dl := imgui.WindowDrawList()

	lt := win.img.lazy.Player0
	ps := win.img.lazy.VCS.TIA.Video.Player0
	if player != 0 {
		lt = win.img.lazy.Player1
		ps = win.img.lazy.VCS.TIA.Video.Player1
	}

	imgui.Spacing()

	// player color indicator. when clicked popup palette is requested. on
	// selection in palette, missile color is changed too
	imguiText("Colour")
	col := lt.Color
	if win.img.imguiSwatch(col) {
		win.popupPalette.request(&col, func() {
			win.img.lazy.Dbg.PushRawEvent(func() { ps.Color = col })

			// update missile color too
			if player != 0 {
				win.img.lazy.Dbg.PushRawEvent(func() { win.img.lazy.VCS.TIA.Video.Missile0.Color = col })
			} else {
				win.img.lazy.Dbg.PushRawEvent(func() { win.img.lazy.VCS.TIA.Video.Missile1.Color = col })
			}
		})
	}

	imguiText("Reflected")
	ref := lt.Reflected
	if imgui.Checkbox("##reflected", &ref) {
		win.img.lazy.Dbg.PushRawEvent(func() { ps.Reflected = ref })
	}

	imgui.SameLine()
	imguiText("Vert. Delay")
	vd := lt.VerticalDelay
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
	hmove := fmt.Sprintf("%01x", lt.Hmove)
	if imguiHexInput("##hmove", !win.img.paused, 1, &hmove) {
		if v, err := strconv.ParseUint(hmove, 16, 8); err == nil {
			win.img.lazy.Dbg.PushRawEvent(func() { ps.Hmove = uint8(v) })
		}
	}
	imgui.PopItemWidth()

	// hmove slider
	imgui.SameLine()
	imgui.PushItemWidth(win.hmoveSliderWidth)
	hmoveSlider := int32(lt.Hmove) - 8
	if imgui.SliderIntV("##hmoveslider", &hmoveSlider, -8, 7, "%d") {
		win.img.lazy.Dbg.PushRawEvent(func() { ps.Hmove = uint8(hmoveSlider + 8) })
	}
	imgui.PopItemWidth()

	imgui.Spacing()
	imgui.Spacing()

	// graphics data - new
	imguiText("New Gfx")
	ngfxSeq := newDrawlistSequence(win.img, imgui.Vec2{X: imgui.FrameHeight(), Y: imgui.FrameHeight()}, 0.1)
	od := lt.GfxDataNew
	for i := 0; i < 8; i++ {
		var col uint8
		if (od<<i)&0x80 == 0x80 {
			col = lt.Color
		} else {
			col = 0x0
			ngfxSeq.nextItemDepressed = true
		}
		if ngfxSeq.rectFilled(col) {
			od ^= 0x80 >> i
			win.img.lazy.Dbg.PushRawEvent(func() { ps.GfxDataNew = od })
		}
		ngfxSeq.sameLine()
	}
	ngfxSeq.end()

	// graphics data - old
	imgui.SameLine()
	imguiText("Old Gfx")
	ogfxSeq := newDrawlistSequence(win.img, imgui.Vec2{X: imgui.FrameHeight(), Y: imgui.FrameHeight()}, 0.1)
	nd := lt.GfxDataOld
	for i := 0; i < 8; i++ {
		var col uint8
		if (nd<<i)&0x80 == 0x80 {
			col = lt.Color
		} else {
			col = 0x0
			ogfxSeq.nextItemDepressed = true
		}
		if ogfxSeq.rectFilled(col) {
			nd ^= 0x80 >> i
			win.img.lazy.Dbg.PushRawEvent(func() { ps.GfxDataOld = nd })
		}
		ogfxSeq.sameLine()
	}
	ogfxSeq.end()

	// scancounter index pointer
	if lt.ScanIsActive {
		var idx int
		if lt.Reflected {
			idx = 7 - lt.ScanPixel
		} else {
			idx = lt.ScanPixel
		}

		seq := ngfxSeq
		if lt.VerticalDelay {
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
	imguiText("NUSIZ")
	imgui.SameLine()
	imgui.PushItemWidth(win.byteDim.X)
	nusiz := fmt.Sprintf("%d", lt.Nusiz)
	if imguiInput("##nusiz", !win.img.paused, 1, &nusiz, "01234567") {
		if v, err := strconv.ParseUint(nusiz, 16, 8); err == nil {
			win.img.lazy.Dbg.PushRawEvent(func() { ps.Nusiz = uint8(v) })
		}
	}
	imgui.PopItemWidth()
	imgui.SameLine()

	// add a note to indicate that the nusiz value is about to update
	if lt.ScanIsActive && lt.Nusiz != lt.ScanLatchedNusiz {
		imguiText("*")
	}

	// interpret nusiz value
	switch lt.Nusiz {
	case 0x0:
		imguiText("one copy")
	case 0x1:
		imguiText("two copies [close]")
	case 0x2:
		imguiText("two copies [med]")
	case 0x3:
		imguiText("three copies [close]")
	case 0x4:
		imguiText("two copies [wide]")
	case 0x5:
		imguiText("double-size")
	case 0x6:
		imguiText("three copies [med]")
	case 0x7:
		imguiText("quad-size")
	default:
		panic("illegal value for player nusiz")
	}

	if (lt.ScanIsActive || lt.ScanIsLatching) &&
		lt.Nusiz != 0x0 && lt.Nusiz != 0x5 && lt.Nusiz != 0x07 {

		if lt.ScanIsActive {
			imguiText("drawing")
		} else {
			imguiText("latching")
		}

		switch lt.ScanCpy {
		case 0:
		case 1:
			imguiText("2nd copy")
		case 2:
			imguiText("3rd copy")
		default:
			panic("more than 2 copies of player!?")
		}
	}
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// horizontal positioning
	imgui.Text(fmt.Sprintf("Last reset at pixel %03d. Draws at pixel %03d", lt.ResetPixel, lt.HmovedPixel))

	if lt.MoreHmove {
		imgui.SameLine()
		imgui.Text("[currently moving]")
	}
}
