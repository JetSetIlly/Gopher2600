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
	"gopher2600/hardware/tia/video"

	"github.com/inkyblackness/imgui-go/v2"
)

func (win *winTIA) drawPlayfield() {
	pf := win.img.vcs.TIA.Video.Playfield

	imgui.Spacing()

	// foreground color indicator. when clicked popup palette is requested. on
	// selection in palette, missile color is changed too
	imgui.BeginGroup()
	imguiText("Foreground")
	if win.img.imguiSwatch(pf.ForegroundColor) {
		win.popupPalette.request(&pf.ForegroundColor, func() {
			// update ball color too
			win.img.vcs.TIA.Video.Ball.Color = pf.ForegroundColor
		})
	}

	// background color indicator. when clicked popup palette is requested
	imguiText("Background")
	if win.img.imguiSwatch(pf.BackgroundColor) {
		win.popupPalette.request(&pf.BackgroundColor, nil)
	}
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// playfield control bits
	imguiText("Reflected")
	imgui.Checkbox("##reflected", &pf.Reflected)
	imgui.SameLine()
	imguiText("Priority")
	imgui.Checkbox("##priority", &pf.Priority)
	imgui.SameLine()
	imguiText("Scoremode")
	imgui.Checkbox("##scoremode", &pf.Scoremode)

	imgui.Spacing()
	imgui.Spacing()

	// playfield data
	imguiText("PF0")
	imgui.SameLine()
	seq := newDrawlistSequence(win.img, imgui.Vec2{X: imgui.FrameHeight(), Y: imgui.FrameHeight()}, 0.1)
	d := pf.PF0
	for i := 0; i < 4; i++ {
		var col uint8
		if (d<<i)&0x80 == 0x80 {
			col = pf.ForegroundColor
		} else {
			col = pf.BackgroundColor
			seq.nextItemDepressed = true
		}
		if seq.rectFilled(col) {
			d ^= 0x80 >> i
			pf.SetPF0(d)
		}
		seq.sameLine()
	}
	seq.end()

	imgui.SameLine()
	imguiText("PF1")
	imgui.SameLine()
	seq.start()
	d = pf.PF1
	for i := 0; i < 8; i++ {
		var col uint8
		if (d<<i)&0x80 == 0x80 {
			col = pf.ForegroundColor
		} else {
			col = pf.BackgroundColor
			seq.nextItemDepressed = true
		}
		if seq.rectFilled(col) {
			d ^= 0x80 >> i
			pf.SetPF1(d)
		}
		seq.sameLine()
	}
	seq.end()

	imgui.SameLine()
	imguiText("PF2")
	imgui.SameLine()
	seq.start()
	d = pf.PF2
	for i := 0; i < 8; i++ {
		var col uint8
		if (d<<i)&0x80 == 0x80 {
			col = pf.ForegroundColor
		} else {
			col = pf.BackgroundColor
			seq.nextItemDepressed = true
		}
		if seq.rectFilled(col) {
			d ^= 0x80 >> i
			pf.SetPF2(d)
		}
		seq.sameLine()
	}
	seq.end()

	imgui.Spacing()
	imgui.Spacing()

	// playfield data as a sequence
	imguiText("Sequence")
	seq = newDrawlistSequence(win.img, imgui.Vec2{X: imgui.FrameHeight() * 0.5, Y: imgui.FrameHeight()}, 0.1)

	// first half of the playfield
	for _, v := range pf.Data {
		var col uint8
		if v {
			col = pf.ForegroundColor
		} else {
			col = pf.BackgroundColor
		}
		seq.rectFilled(col)
		seq.sameLine()
	}

	// second half of the playfield
	for i, v := range pf.Data {
		// correct for reflected playfield
		if pf.Reflected {
			v = pf.Data[len(pf.Data)-1-i]
		}

		var col uint8
		if v {
			col = pf.ForegroundColor
		} else {
			col = pf.BackgroundColor
		}
		seq.rectFilled(col)
		seq.sameLine()
	}
	seq.end()

	// playfield index pointer
	if pf.Region != video.RegionOffScreen {
		idx := pf.Idx
		if pf.Region == video.RegionRight {
			idx += len(pf.Data)
		}

		p1 := imgui.Vec2{
			X: seq.offsetX(idx),
			Y: imgui.CursorScreenPos().Y,
		}

		dl := imgui.WindowDrawList()
		dl.AddCircleFilled(p1, imgui.FontSize()*0.20, win.idxPointer)
	}
}
