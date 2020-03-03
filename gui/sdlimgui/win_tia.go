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

const winTIATitle = "TIA"

type winTIA struct {
	windowManagement
	img              *SdlImgui
	popupPalette     *popupPalette
	playfieldPointer imgui.PackedColor
}

func newWinTIA(img *SdlImgui) (managedWindow, error) {
	win := &winTIA{
		img:          img,
		popupPalette: newPopupPalette(img),
	}

	return win, nil
}

func (win *winTIA) init() {
	win.playfieldPointer = imgui.PackedColorFromVec4(win.img.cols.PlayfieldPointer)
}

func (win *winTIA) destroy() {
}

func (win *winTIA) id() string {
	return winTIATitle
}

// draw is called by service loop
func (win *winTIA) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{12, 500}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{718, 156}, imgui.ConditionFirstUseEver)
	imgui.BeginV(winTIATitle, &win.open, 0)

	imgui.BeginTabBar("")
	if imgui.BeginTabItem("Playfield") {
		win.drawPlayfield()
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Player 0") {
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Player 1") {
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Missile 0") {
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Missile 1") {
		imgui.EndTabItem()
	}
	if imgui.BeginTabItem("Ball") {
		imgui.EndTabItem()
	}
	imgui.EndTabBar()

	imgui.End()

	win.popupPalette.draw()
}

func (win *winTIA) drawPlayfield() {
	pf := win.img.vcs.TIA.Video.Playfield

	imgui.Spacing()

	imgui.BeginGroup()
	imguiLabel("Foreground")
	if win.img.imguiSwatch(pf.ForegroundColor) {
		win.popupPalette.request(&pf.ForegroundColor)
	}

	imguiLabel("Background")
	if win.img.imguiSwatch(pf.BackgroundColor) {
		win.popupPalette.request(&pf.BackgroundColor)
	}
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	imguiLabel("Reflected")
	imgui.Checkbox("##reflected", &pf.Reflected)

	imgui.SameLine()
	imguiLabel("Priority")
	imgui.Checkbox("##priority", &pf.Priority)

	imgui.SameLine()
	imguiLabel("Scoremode")
	imgui.Checkbox("##scoremode", &pf.Scoremode)

	imgui.Spacing()
	imgui.Spacing()
	imgui.Spacing()

	imguiLabel("PF0")
	imgui.SameLine()
	imgui.BeginGroup()
	seq := newDrawlistSequence(win.img, imgui.Vec2{X: imgui.FrameHeight(), Y: imgui.FrameHeight()}, 0.1)
	for i := 0; i < 4; i++ {
		var col uint8
		if (pf.PF0<<i)&0x80 == 0x80 {
			col = pf.ForegroundColor
		} else {
			col = pf.BackgroundColor
		}
		if seq.rectFilled(col) {
			v := pf.PF0
			v ^= 0x80 >> i
			pf.SetPF0(v)
		}
		seq.sameLine()
	}
	imgui.EndGroup()

	imgui.SameLine()
	imguiLabel("PF1")
	imgui.SameLine()
	imgui.BeginGroup()
	seq.start()
	for i := 0; i < 8; i++ {
		var col uint8
		if (pf.PF1<<i)&0x80 == 0x80 {
			col = pf.ForegroundColor
		} else {
			col = pf.BackgroundColor
		}
		if seq.rectFilled(col) {
			v := pf.PF1
			v ^= 0x80 >> i
			pf.SetPF1(v)
		}
		seq.sameLine()
	}
	imgui.EndGroup()

	imgui.SameLine()
	imguiLabel("PF2")
	imgui.SameLine()
	imgui.BeginGroup()
	seq.start()
	for i := 0; i < 8; i++ {
		var col uint8
		if (pf.PF2<<i)&0x80 == 0x80 {
			col = pf.ForegroundColor
		} else {
			col = pf.BackgroundColor
		}
		if seq.rectFilled(col) {
			v := pf.PF2
			v ^= 0x80 >> i
			pf.SetPF2(v)
		}
		seq.sameLine()
	}
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	imguiLabel("Sequence")
	imgui.BeginGroup()
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
	imgui.EndGroup()

	// playfield pointer
	if pf.Region != video.RegionOffScreen {
		idx := pf.Idx
		if pf.Region == video.RegionRight {
			idx += len(pf.Data)
		}

		p1 := imgui.Vec2{
			X: seq.offsetX(idx),
			Y: imgui.WindowPos().Y + imgui.CursorPosY() + 2.0,
		}

		dl := imgui.WindowDrawList()
		dl.AddCircleFilled(p1, seq.size.X*0.25, win.playfieldPointer)
	}
}
