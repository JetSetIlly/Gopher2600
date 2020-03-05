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

	ps := win.img.vcs.TIA.Video.Player0
	if player != 0 {
		ps = win.img.vcs.TIA.Video.Player1
	}

	imgui.Spacing()

	// player color indicator. when clicked popup palette is requested. on
	// selection in palette, missile color is changed too
	imguiLabel("Colour")
	if win.img.imguiSwatch(ps.Color) {
		win.popupPalette.request(&ps.Color, func() {
			// update missile color too
			if player != 0 {
				win.img.vcs.TIA.Video.Missile0.Color = ps.Color
			} else {
				win.img.vcs.TIA.Video.Missile1.Color = ps.Color
			}
		})
	}

	imguiLabel("Reflected")
	imgui.Checkbox("##reflected", &ps.Reflected)

	imgui.SameLine()
	imguiLabel("Vert. Delay")
	v := ps.VerticalDelay
	if imgui.Checkbox("##vertdelay", &v) {
		// vertical delay affects which gfx register to use. set vertical delay
		// using the SetVerticalDelay function
		ps.SetVerticalDelay(v)
	}

	imgui.Spacing()
	imgui.Spacing()

	// hmove value
	imguiLabel("HMOVE")
	imgui.SameLine()
	imgui.PushItemWidth(win.byteDim.X)
	hmove := fmt.Sprintf("%01x", ps.Hmove)
	if imguiHexInput("##hmove", !win.img.paused, 1, &hmove) {
		if v, err := strconv.ParseUint(hmove, 16, 8); err == nil {
			ps.Hmove = uint8(v)
		}
	}
	imgui.PopItemWidth()

	// hmove slider
	imgui.SameLine()
	imgui.PushItemWidth(win.hmoveSliderWidth)
	hmoveSlider := int32(ps.Hmove) - 8
	if imgui.SliderIntV("##hmoveslider", &hmoveSlider, -8, 7, "%d") {
		ps.Hmove = uint8(hmoveSlider + 8)
	}
	imgui.PopItemWidth()

	imgui.Spacing()
	imgui.Spacing()

	// graphics data - new
	imguiLabel("New Gfx")
	ngfxSeq := newDrawlistSequence(win.img, imgui.Vec2{X: imgui.FrameHeight(), Y: imgui.FrameHeight()}, 0.1)
	d := ps.GfxDataNew
	for i := 0; i < 8; i++ {
		var col uint8
		if (d<<i)&0x80 == 0x80 {
			col = ps.Color
		} else {
			col = 0x0
			ngfxSeq.nextItemDepressed = true
		}
		if ngfxSeq.rectFilled(col) {
			d ^= 0x80 >> i
			ps.GfxDataNew = d
		}
		ngfxSeq.sameLine()
	}
	ngfxSeq.end()

	// graphics data - old
	imgui.SameLine()
	imguiLabel("Old Gfx")
	ogfxSeq := newDrawlistSequence(win.img, imgui.Vec2{X: imgui.FrameHeight(), Y: imgui.FrameHeight()}, 0.1)
	d = ps.GfxDataOld
	for i := 0; i < 8; i++ {
		var col uint8
		if (d<<i)&0x80 == 0x80 {
			col = ps.Color
		} else {
			col = 0x0
			ogfxSeq.nextItemDepressed = true
		}
		if ogfxSeq.rectFilled(col) {
			d ^= 0x80 >> i
			ps.GfxDataOld = d
		}
		ogfxSeq.sameLine()
	}
	ogfxSeq.end()

	// scancounter index pointer
	if ps.ScanCounter.IsActive() {
		var idx int
		if ps.Reflected {
			idx = 7 - ps.ScanCounter.Pixel
		} else {
			idx = ps.ScanCounter.Pixel
		}

		seq := ngfxSeq
		if ps.VerticalDelay {
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
	imguiLabel("NUSIZ")
	imgui.SameLine()
	imgui.PushItemWidth(win.byteDim.X)
	nusiz := fmt.Sprintf("%d", ps.Nusiz)
	if imguiInput("##nusiz", !win.img.paused, 1, &nusiz, "01234567") {
		if v, err := strconv.ParseUint(nusiz, 16, 8); err == nil {
			ps.Nusiz = uint8(v)
		}
	}
	imgui.PopItemWidth()
	imgui.SameLine()

	// add a note to indicate that the nusiz value is about to update
	if ps.ScanCounter.IsActive() && ps.Nusiz != ps.ScanCounter.LatchedNusiz {
		imguiLabel("*")
	}

	// interpret nusiz value
	switch ps.Nusiz {
	case 0x0:
		imguiLabel("one copy")
	case 0x1:
		imguiLabel("two copies [close]")
	case 0x2:
		imguiLabel("two copies [med]")
	case 0x3:
		imguiLabel("three copies [close]")
	case 0x4:
		imguiLabel("two copies [wide]")
	case 0x5:
		imguiLabel("double-size")
	case 0x6:
		imguiLabel("three copies [med]")
	case 0x7:
		imguiLabel("quad-size")
	default:
		panic("illegal value for player nusiz")
	}

	if (ps.ScanCounter.IsActive() || ps.ScanCounter.IsLatching()) &&
		ps.Nusiz != 0x0 && ps.Nusiz != 0x5 && ps.Nusiz != 0x07 {

		if ps.ScanCounter.IsActive() {
			imguiLabel("drawing")
		} else {
			imguiLabel("latching")
		}

		switch ps.ScanCounter.Cpy {
		case 0:
		case 1:
			imguiLabel("2nd copy")
		case 2:
			imguiLabel("3rd copy")
		default:
			panic("more than 2 copies of player!?")
		}
	}
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// horizontal positioning
	imgui.Text(fmt.Sprintf("Last reset at pixel %03d. Draws at pixel %03d", ps.ResetPixel, ps.HmovedPixel))

	if ps.MoreHMOVE {
		imgui.SameLine()
		imgui.Text("[currently moving]")
	}
}
