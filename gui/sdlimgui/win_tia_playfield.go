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

	"github.com/jetsetilly/gopher2600/hardware/tia/video"
	"github.com/jetsetilly/imgui-go/v5"
)

func (win *winTIA) drawPlayfield() {
	playfield := win.img.cache.VCS.TIA.Video.Playfield
	player0 := win.img.cache.VCS.TIA.Video.Player0
	player1 := win.img.cache.VCS.TIA.Video.Player1

	imgui.Spacing()

	// foreground color indicator. when clicked popup palette is requested. on
	// selection in palette, missile color is changed too
	imgui.BeginGroup()

	imguiLabel("Foreground")
	fgCol := playfield.ForegroundColor
	if win.img.imguiTVColourSwatch(fgCol, 0.75) {
		win.popupPalette.request(&fgCol, func() {
			win.img.dbg.PushFunction(func() {
				realPlayfield := win.img.dbg.VCS().TIA.Video.Playfield
				realPlayfield.ForegroundColor = fgCol
				realBall := win.img.dbg.VCS().TIA.Video.Ball
				realBall.Color = fgCol
			})
		})
	}

	imgui.SameLine()

	// background color indicator. when clicked popup palette is requested
	imguiLabel("Background")
	bgCol := playfield.BackgroundColor
	if win.img.imguiTVColourSwatch(bgCol, 0.75) {
		win.popupPalette.request(&bgCol, func() {
			win.img.dbg.PushFunction(func() {
				realPlayfield := win.img.dbg.VCS().TIA.Video.Playfield
				realPlayfield.BackgroundColor = bgCol
			})
		})
	}
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// playfield control bits
	imgui.BeginGroup()
	imguiLabel("Reflected")
	ref := playfield.Reflected
	if imgui.Checkbox("##reflected", &ref) {
		win.img.dbg.PushFunction(func() {
			realPlayfield := win.img.dbg.VCS().TIA.Video.Playfield
			realPlayfield.Reflected = ref
		})
	}
	imgui.SameLine()
	imguiLabel("Scoremode")
	sm := playfield.Scoremode
	if imgui.Checkbox("##scoremode", &sm) {
		win.img.dbg.PushFunction(func() {
			realPlayfield := win.img.dbg.VCS().TIA.Video.Playfield
			realPlayfield.Scoremode = sm
		})
	}
	imgui.SameLine()
	imguiLabel("Priority")
	pri := playfield.Priority
	if imgui.Checkbox("##priority", &pri) {
		win.img.dbg.PushFunction(func() {
			realPlayfield := win.img.dbg.VCS().TIA.Video.Playfield
			realPlayfield.Priority = pri
		})
	}

	imgui.SameLine()
	imguiLabel("CTRLPF")
	imgui.SameLine()
	ctrlpf := fmt.Sprintf("%02x", playfield.Ctrlpf)
	if imguiHexInput("##ctrlpf", 2, &ctrlpf) {
		if v, err := strconv.ParseUint(ctrlpf, 16, 8); err == nil {
			win.img.dbg.PushFunction(func() {
				// update ball copy of CTRLPF too in addition to the playfield copy
				realPlayfield := win.img.dbg.VCS().TIA.Video.Playfield
				realPlayfield.SetCTRLPF(uint8(v))
				realBall := win.img.dbg.VCS().TIA.Video.Ball
				realBall.SetCTRLPF(uint8(v))
			})
		}
	}
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// playfield data
	imgui.BeginGroup()
	imguiLabel("PF0")
	imgui.SameLine()
	seq := newDrawlistSequence(imgui.Vec2{X: imgui.FrameHeight(), Y: imgui.FrameHeight()}, false)
	pf0d := playfield.PF0
	for i := 0; i < 4; i++ {
		var col uint8
		if (pf0d<<i)&0x80 == 0x80 {
			col = playfield.ForegroundColor
		} else {
			col = playfield.BackgroundColor
			seq.nextItemDepressed = true
		}
		if seq.rectFill(win.img.getTVColour(col)) {
			pf0d ^= 0x80 >> i
			win.img.dbg.PushFunction(func() {
				realPlayfield := win.img.dbg.VCS().TIA.Video.Playfield
				realPlayfield.SetPF0(pf0d)
			})
		}
		seq.sameLine()
	}
	seq.end()

	imgui.SameLine()
	imguiLabel("PF1")
	imgui.SameLine()
	seq.start()
	pf1d := playfield.PF1
	for i := 0; i < 8; i++ {
		var col uint8
		if (pf1d<<i)&0x80 == 0x80 {
			col = playfield.ForegroundColor
		} else {
			col = playfield.BackgroundColor
			seq.nextItemDepressed = true
		}
		if seq.rectFill(win.img.getTVColour(col)) {
			pf1d ^= 0x80 >> i
			win.img.dbg.PushFunction(func() {
				realPlayfield := win.img.dbg.VCS().TIA.Video.Playfield
				realPlayfield.SetPF1(pf1d)
			})
		}
		seq.sameLine()
	}
	seq.end()

	imgui.SameLine()
	imguiLabel("PF2")
	imgui.SameLine()
	seq.start()
	pf2d := playfield.PF2
	for i := 0; i < 8; i++ {
		var col uint8
		if (pf2d<<i)&0x80 == 0x80 {
			col = playfield.ForegroundColor
		} else {
			col = playfield.BackgroundColor
			seq.nextItemDepressed = true
		}
		if seq.rectFill(win.img.getTVColour(col)) {
			pf2d ^= 0x80 >> i
			win.img.dbg.PushFunction(func() {
				realPlayfield := win.img.dbg.VCS().TIA.Video.Playfield
				realPlayfield.SetPF2(pf2d)
			})
		}
		seq.sameLine()
	}
	seq.end()

	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// playfield data for the scanline
	imgui.BeginGroup()
	imguiLabel("Scanline")

	seq = newDrawlistSequence(imgui.Vec2{X: imgui.FrameHeight() * 0.5, Y: imgui.FrameHeight()}, false)

	// first half of the playfield
	for _, v := range *playfield.LeftData {
		var col uint8
		if v {
			if playfield.Scoremode {
				col = player0.Color
			} else {
				col = playfield.ForegroundColor
			}
		} else {
			col = playfield.BackgroundColor
		}
		seq.rectFill(win.img.getTVColour(col))
		seq.sameLine()
	}

	// second half of the playfield
	for _, v := range *playfield.RightData {
		var col uint8
		if v {
			if playfield.Scoremode {
				col = player1.Color
			} else {
				col = playfield.ForegroundColor
			}
		} else {
			col = playfield.BackgroundColor
		}
		seq.rectFill(win.img.getTVColour(col))
		seq.sameLine()
	}
	seq.end()

	// playfield index pointer
	if playfield.Region != video.RegionOffScreen {
		idx := playfield.Idx
		if playfield.Region == video.RegionRight {
			idx += len(*playfield.LeftData)
		}

		p1 := imgui.Vec2{
			X: seq.offsetX(idx),
			Y: imgui.CursorScreenPos().Y,
		}

		dl := imgui.WindowDrawList()
		dl.AddCircleFilled(p1, imgui.FontSize()*0.20, win.img.cols.tiaPointer)
	}
	imgui.EndGroup()

	if playfield.Scoremode {
		imgui.Spacing()
		imgui.Spacing()
		imgui.Text("(the scoremode flag affects the color of the playfield)")
	}
}
