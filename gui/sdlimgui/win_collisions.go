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
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/hardware/tia/video"
)

const winCollisionsID = "Collisions"

type winCollisions struct {
	img  *SdlImgui
	open bool
}

func newWinCollisions(img *SdlImgui) (window, error) {
	win := &winCollisions{
		img: img,
	}

	return win, nil
}

func (win *winCollisions) init() {
}

func (win *winCollisions) id() string {
	return winCollisionsID
}

func (win *winCollisions) isOpen() bool {
	return win.open
}

func (win *winCollisions) setOpen(open bool) {
	win.open = open
}

func (win *winCollisions) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{530, 455}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)

	imguiLabel("CXM0P ")
	win.drawCollision(win.img.lz.Collisions.CXM0P, &win.img.lz.Dbg.VCS.TIA.Video.Collisions.CXM0P, video.CollisionMask)
	imguiLabel("CXM1P ")
	win.drawCollision(win.img.lz.Collisions.CXM1P, &win.img.lz.Dbg.VCS.TIA.Video.Collisions.CXM1P, video.CollisionMask)
	imguiLabel("CXP0FB")
	win.drawCollision(win.img.lz.Collisions.CXP0FB, &win.img.lz.Dbg.VCS.TIA.Video.Collisions.CXP0FB, video.CollisionMask)
	imguiLabel("CXP1FB")
	win.drawCollision(win.img.lz.Collisions.CXP1FB, &win.img.lz.Dbg.VCS.TIA.Video.Collisions.CXP1FB, video.CollisionMask)
	imguiLabel("CXM0FB")
	win.drawCollision(win.img.lz.Collisions.CXM0FB, &win.img.lz.Dbg.VCS.TIA.Video.Collisions.CXM0FB, video.CollisionMask)
	imguiLabel("CXM1FB")
	win.drawCollision(win.img.lz.Collisions.CXM1FB, &win.img.lz.Dbg.VCS.TIA.Video.Collisions.CXM1FB, video.CollisionMask)
	imguiLabel("CXBLPF")
	win.drawCollision(win.img.lz.Collisions.CXBLPF, &win.img.lz.Dbg.VCS.TIA.Video.Collisions.CXBLPF, video.CollisionCXBLPFMask)
	imguiLabel("CXPPMM")
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
	drawCollision(win.img, read, mask,
		func(b uint8) {
			win.img.lz.Dbg.PushRawEvent(func() {
				*write = b
			})
		})
}

// drawCollision() is used by the dbgscr tooltip for the collision layer.
func drawCollision(img *SdlImgui, value uint8, mask uint8, onWrite func(uint8)) {
	seq := newDrawlistSequence(img, imgui.Vec2{X: imgui.FrameHeight() * 0.75, Y: imgui.FrameHeight() * 0.75}, false)
	for i := 0; i < 8; i++ {
		if mask<<i&0x80 == 0x80 {
			if (value<<i)&0x80 != 0x80 {
				seq.nextItemDepressed = true
			}
			if seq.rectFill(img.cols.collisionBit) {
				b := value ^ (0x80 >> i)
				onWrite(b)
			}
		} else {
			seq.nextItemDepressed = true
			seq.rectEmpty(img.cols.collisionBit)
		}
		seq.sameLine()
	}
	seq.end()
}
