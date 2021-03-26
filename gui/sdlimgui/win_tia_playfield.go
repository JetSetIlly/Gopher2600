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

	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/tia/video"

	"github.com/inkyblackness/imgui-go/v4"
)

func (win *winTIA) drawPlayfield() {
	imgui.BeginChildV("##playfield", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.scopeHeight}, false, 0)
	defer imgui.EndChild()

	lz := win.img.lz.Playfield
	pf := win.img.lz.Playfield.Pf
	bs := win.img.lz.Ball.Bs

	imgui.Spacing()

	// foreground color indicator. when clicked popup palette is requested. on
	// selection in palette, missile color is changed too
	imgui.BeginGroup()

	imguiLabel("Foreground")
	fgCol := lz.ForegroundColor
	if win.img.imguiSwatch(fgCol, 0.75) {
		win.popupPalette.request(&fgCol, func() {
			win.update(
				func() {
					win.img.lz.Dbg.PushDeepPoke(addresses.WriteAddress["COLUPF"], pf.ForegroundColor, fgCol, 0xfe)
				},
				func() {
					pf.ForegroundColor = fgCol
					bs.Color = fgCol
				})
		})
	}

	// background color indicator. when clicked popup palette is requested
	imguiLabel("Background")
	bgCol := lz.BackgroundColor
	if win.img.imguiSwatch(bgCol, 0.75) {
		win.popupPalette.request(&bgCol, func() {
			win.update(
				func() {
					win.img.lz.Dbg.PushDeepPoke(addresses.WriteAddress["COLUBK"], pf.BackgroundColor, bgCol, 0xfe)
				},
				func() {
					pf.BackgroundColor = bgCol
				})
		})
	}
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// playfield control bits
	imgui.BeginGroup()
	imguiLabel("Reflected")
	ref := lz.Reflected
	if imgui.Checkbox("##reflected", &ref) {
		win.update(
			func() {
				var o uint8
				if pf.Reflected {
					o = addresses.CTRLPFReflectedMask
				}
				var n uint8
				if ref {
					n = addresses.CTRLPFReflectedMask
				}
				win.img.lz.Dbg.PushDeepPoke(addresses.WriteAddress["CTRLPF"], o, n, addresses.CTRLPFReflectedMask)
			},
			func() {
				pf.Reflected = ref
			})
	}
	imgui.SameLine()
	imguiLabel("Scoremode")
	sm := lz.Scoremode
	if imgui.Checkbox("##scoremode", &sm) {
		win.update(
			func() {
				var o uint8
				if pf.Scoremode {
					o = addresses.CTRLPFScoremodeMask
				}
				var n uint8
				if sm {
					n = addresses.CTRLPFScoremodeMask
				}
				win.img.lz.Dbg.PushDeepPoke(addresses.WriteAddress["CTRLPF"], o, n, addresses.CTRLPFScoremodeMask)
			},
			func() {
				pf.Scoremode = sm
			})
	}
	imgui.SameLine()
	imguiLabel("Priority")
	pri := lz.Priority
	if imgui.Checkbox("##priority", &pri) {
		win.update(
			func() {
				var o uint8
				if pf.Priority {
					o = addresses.CTRLPFPriorityMask
				}
				var n uint8
				if pri {
					n = addresses.CTRLPFPriorityMask
				}
				win.img.lz.Dbg.PushDeepPoke(addresses.WriteAddress["CTRLPF"], o, n, addresses.CTRLPFPriorityMask)
			},
			func() {
				pf.Priority = pri
			})
	}

	imgui.SameLine()
	imguiLabel("CTRLPF")
	imgui.SameLine()
	ctrlpf := fmt.Sprintf("%02x", lz.Ctrlpf)
	if imguiHexInput("##ctrlpf", 2, &ctrlpf) {
		if v, err := strconv.ParseUint(ctrlpf, 16, 8); err == nil {
			win.update(
				func() {
					win.img.lz.Dbg.PushDeepPoke(addresses.WriteAddress["CTRLPF"], pf.Ctrlpf, uint8(v), 0xff)
				},
				func() {
					// update ball copy of CTRLPF too in addition to the playfield copy
					pf.SetCTRLPF(uint8(v))
					bs.SetCTRLPF(uint8(v))
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
	seq := newDrawlistSequence(win.img, imgui.Vec2{X: imgui.FrameHeight(), Y: imgui.FrameHeight()}, false)
	pf0d := lz.PF0
	for i := 0; i < 4; i++ {
		var col uint8
		if (pf0d<<i)&0x80 == 0x80 {
			col = lz.ForegroundColor
		} else {
			col = lz.BackgroundColor
			seq.nextItemDepressed = true
		}
		if seq.rectFillTvCol(col) {
			pf0d ^= 0x80 >> i
			win.update(
				func() {
					win.img.lz.Dbg.PushDeepPoke(addresses.WriteAddress["PF0"], pf.PF0, pf0d, 0xff)
				},
				func() {
					pf.SetPF0(pf0d)
				})
		}
		seq.sameLine()
	}
	seq.end()

	imgui.SameLine()
	imguiLabel("PF1")
	imgui.SameLine()
	seq.start()
	pf1d := lz.PF1
	for i := 0; i < 8; i++ {
		var col uint8
		if (pf1d<<i)&0x80 == 0x80 {
			col = lz.ForegroundColor
		} else {
			col = lz.BackgroundColor
			seq.nextItemDepressed = true
		}
		if seq.rectFillTvCol(col) {
			pf1d ^= 0x80 >> i
			win.update(
				func() {
					win.img.lz.Dbg.PushDeepPoke(addresses.WriteAddress["PF1"], pf.PF1, pf1d, 0xff)
				},
				func() {
					pf.SetPF1(pf1d)
				})
		}
		seq.sameLine()
	}
	seq.end()

	imgui.SameLine()
	imguiLabel("PF2")
	imgui.SameLine()
	seq.start()
	pf2d := lz.PF2
	for i := 0; i < 8; i++ {
		var col uint8
		if (pf2d<<i)&0x80 == 0x80 {
			col = lz.ForegroundColor
		} else {
			col = lz.BackgroundColor
			seq.nextItemDepressed = true
		}
		if seq.rectFillTvCol(col) {
			pf2d ^= 0x80 >> i
			win.update(
				func() {
					win.img.lz.Dbg.PushDeepPoke(addresses.WriteAddress["PF2"], pf.PF2, pf2d, 0xff)
				},
				func() {
					pf.SetPF2(pf2d)
				})
		}
		seq.sameLine()
	}
	seq.end()
	imgui.EndGroup()

	imgui.Spacing()
	imgui.Spacing()

	// playfield data as a sequence
	imgui.BeginGroup()
	imguiLabel("Sequence")
	seq = newDrawlistSequence(win.img, imgui.Vec2{X: imgui.FrameHeight() * 0.5, Y: imgui.FrameHeight()}, false)

	// first half of the playfield
	for _, v := range lz.LeftData {
		var col uint8
		if v {
			col = lz.ForegroundColor
		} else {
			col = lz.BackgroundColor
		}
		seq.rectFillTvCol(col)
		seq.sameLine()
	}

	// second half of the playfield
	for _, v := range lz.RightData {
		var col uint8
		if v {
			col = lz.ForegroundColor
		} else {
			col = lz.BackgroundColor
		}
		seq.rectFillTvCol(col)
		seq.sameLine()
	}
	seq.end()

	// playfield index pointer
	if lz.Region != video.RegionOffScreen {
		idx := lz.Idx
		if lz.Region == video.RegionRight {
			idx += len(lz.LeftData)
		}

		p1 := imgui.Vec2{
			X: seq.offsetX(idx),
			Y: imgui.CursorScreenPos().Y,
		}

		dl := imgui.WindowDrawList()
		dl.AddCircleFilled(p1, imgui.FontSize()*0.20, win.img.cols.tiaPointer)
	}
	imgui.EndGroup()
}
