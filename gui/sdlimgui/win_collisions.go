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
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/memory/vcs"
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
	defer imgui.End()

	imguiLabel("CXM0P ")
	drawRegister("##CXM0P", win.img.lz.Collisions.CXM0P, vcs.TIADrivenPins, win.img.cols.collisionBit,
		func(v uint8) {
			win.img.dbg.PushRawEvent(func() {
				win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.CXM0P], v)
			})
		})

	imguiLabel("CXM1P ")
	drawRegister("##CXM1P", win.img.lz.Collisions.CXM1P, vcs.TIADrivenPins, win.img.cols.collisionBit,
		func(v uint8) {
			win.img.dbg.PushRawEvent(func() {
				win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.CXM1P], v)
			})
		})

	imguiLabel("CXP0FB")
	drawRegister("##CXP0FB", win.img.lz.Collisions.CXP0FB, vcs.TIADrivenPins, win.img.cols.collisionBit,
		func(v uint8) {
			win.img.dbg.PushRawEvent(func() {
				win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.CXP0FB], v)
			})
		})

	imguiLabel("CXP1FB")
	drawRegister("##CXP1FB", win.img.lz.Collisions.CXP1FB, vcs.TIADrivenPins, win.img.cols.collisionBit,
		func(v uint8) {
			win.img.dbg.PushRawEvent(func() {
				win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.CXP1FB], v)
			})
		})

	imguiLabel("CXM0FB")
	drawRegister("##CXM0FB", win.img.lz.Collisions.CXM0FB, vcs.TIADrivenPins, win.img.cols.collisionBit,
		func(v uint8) {
			win.img.dbg.PushRawEvent(func() {
				win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.CXM0FB], v)
			})
		})

	imguiLabel("CXM1FB")
	drawRegister("##CXM1FB", win.img.lz.Collisions.CXM1FB, vcs.TIADrivenPins, win.img.cols.collisionBit,
		func(v uint8) {
			win.img.dbg.PushRawEvent(func() {
				win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.CXM1FB], v)
			})
		})

	imguiLabel("CXBLPF")
	drawRegister("##CXBLPF", win.img.lz.Collisions.CXBLPF, vcs.TIADrivenPins, win.img.cols.collisionBit,
		func(v uint8) {
			win.img.dbg.PushRawEvent(func() {
				win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.CXBLPF], v)
			})
		})

	imguiLabel("CXPPMM")
	drawRegister("##CXPPMM", win.img.lz.Collisions.CXPPMM, vcs.TIADrivenPins, win.img.cols.collisionBit,
		func(v uint8) {
			win.img.dbg.PushRawEvent(func() {
				win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.CXPPMM], v)
			})
		})

	imgui.Spacing()

	if imgui.Button("Clear Collisions") {
		win.img.dbg.PushRawEvent(func() {
			win.img.vcs.TIA.Video.Collisions.Clear()
		})
	}
}
