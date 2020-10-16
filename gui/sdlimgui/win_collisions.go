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
	"github.com/inkyblackness/imgui-go/v2"
	"github.com/jetsetilly/gopher2600/hardware/tia/video"
)

const winCollisionsTitle = "Collisions"

type winCollisions struct {
	windowManagement

	img *SdlImgui

	collisionBit imgui.PackedColor
}

func newWinCollisions(img *SdlImgui) (managedWindow, error) {
	win := &winCollisions{
		img: img,
	}

	return win, nil
}

func (win *winCollisions) init() {
	win.collisionBit = imgui.PackedColorFromVec4(win.img.cols.CollisionBit)
}

func (win *winCollisions) destroy() {
}

func (win *winCollisions) id() string {
	return winCollisionsTitle
}

func (win *winCollisions) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{623, 527}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winCollisionsTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	imgui.Text("CXM0P ")
	imgui.SameLine()
	win.drawCollision(win.img.lz.Collisions.CXM0P, &win.img.lz.Dbg.VCS.TIA.Video.Collisions.CXM0P, video.CollisionMask)

	imgui.Text("CXM1P ")
	imgui.SameLine()
	win.drawCollision(win.img.lz.Collisions.CXM1P, &win.img.lz.Dbg.VCS.TIA.Video.Collisions.CXM1P, video.CollisionMask)

	imgui.Text("CXP0FB")
	imgui.SameLine()
	win.drawCollision(win.img.lz.Collisions.CXP0FB, &win.img.lz.Dbg.VCS.TIA.Video.Collisions.CXP0FB, video.CollisionMask)

	imgui.Text("CXP1FB")
	imgui.SameLine()
	win.drawCollision(win.img.lz.Collisions.CXP1FB, &win.img.lz.Dbg.VCS.TIA.Video.Collisions.CXP1FB, video.CollisionMask)

	imgui.Text("CXM0FB")
	imgui.SameLine()
	win.drawCollision(win.img.lz.Collisions.CXM0FB, &win.img.lz.Dbg.VCS.TIA.Video.Collisions.CXM0FB, video.CollisionMask)

	imgui.Text("CXM1FB")
	imgui.SameLine()
	win.drawCollision(win.img.lz.Collisions.CXM1FB, &win.img.lz.Dbg.VCS.TIA.Video.Collisions.CXM1FB, video.CollisionMask)

	imgui.Text("CXBLPF")
	imgui.SameLine()
	win.drawCollision(win.img.lz.Collisions.CXBLPF, &win.img.lz.Dbg.VCS.TIA.Video.Collisions.CXBLPF, video.CollisionCXBLPFMask)

	imgui.Text("CXPPMM")
	imgui.SameLine()
	win.drawCollision(win.img.lz.Collisions.CXPPMM, &win.img.lz.Dbg.VCS.TIA.Video.Collisions.CXPPMM, video.CollisionMask)

	imgui.Spacing()

	if imgui.Button("Clear Collisions") {
		win.img.lz.Dbg.PushRawEvent(func() {
			win.img.lz.Dbg.VCS.TIA.Video.Collisions.Clear()
		})
	}

	imgui.End()
}

func (win *winCollisions) drawCollision(read uint8, write *uint8, mask uint8) {
	seq := newDrawlistSequence(win.img, imgui.Vec2{X: imgui.FrameHeight() * 0.75, Y: imgui.FrameHeight() * 0.75}, false)
	for i := 0; i < 8; i++ {
		if mask<<i&0x80 == 0x80 {
			if (read<<i)&0x80 != 0x80 {
				seq.nextItemDepressed = true
			}
			if seq.rectFill(win.collisionBit) {
				b := read ^ (0x80 >> i)
				win.img.lz.Dbg.PushRawEvent(func() {
					*write = b
				})
			}
		} else {
			seq.nextItemDepressed = true
			seq.rectEmpty(win.collisionBit)
		}
		seq.sameLine()
	}
	seq.end()
}
