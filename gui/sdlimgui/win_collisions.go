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
	debuggerWin

	img *SdlImgui
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

func (win *winCollisions) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 530, Y: 455}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCollisions) draw() {
	cxm0p, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddress[cpubus.CXM0P])
	cxm1p, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddress[cpubus.CXM1P])
	cxp0fb, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddress[cpubus.CXP0FB])
	cxp1fb, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddress[cpubus.CXP1FB])
	cxm0fb, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddress[cpubus.CXM0FB])
	cxm1fb, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddress[cpubus.CXM1FB])
	cxblpf, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddress[cpubus.CXBLPF])
	cxppmm, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddress[cpubus.CXPPMM])

	if imgui.BeginTableV("##collisions", 2, imgui.TableFlagsNone, imgui.Vec2{}, 0.0) {
		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.AlignTextToFramePadding()
		imgui.Text("CXM0P ")
		imgui.TableNextColumn()
		drawRegister("##CXM0P", cxm0p, vcs.TIADrivenPins, win.img.cols.collisionBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.dbg.VCS().Mem.Poke(cpubus.ReadAddress[cpubus.CXM0P], v)
				})
			})

		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.AlignTextToFramePadding()
		imgui.Text("CXM1P ")
		imgui.TableNextColumn()
		drawRegister("##CXM1P", cxm1p, vcs.TIADrivenPins, win.img.cols.collisionBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.dbg.VCS().Mem.Poke(cpubus.ReadAddress[cpubus.CXM1P], v)
				})
			})

		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.AlignTextToFramePadding()
		imgui.Text("CXP0FB")
		imgui.TableNextColumn()
		drawRegister("##CXP0FB", cxp0fb, vcs.TIADrivenPins, win.img.cols.collisionBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.dbg.VCS().Mem.Poke(cpubus.ReadAddress[cpubus.CXP0FB], v)
				})
			})

		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.AlignTextToFramePadding()
		imgui.Text("CXP1FB")
		imgui.TableNextColumn()
		drawRegister("##CXP1FB", cxp1fb, vcs.TIADrivenPins, win.img.cols.collisionBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.dbg.VCS().Mem.Poke(cpubus.ReadAddress[cpubus.CXP1FB], v)
				})
			})

		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.AlignTextToFramePadding()
		imgui.Text("CXM0FB")
		imgui.TableNextColumn()
		drawRegister("##CXM0FB", cxm0fb, vcs.TIADrivenPins, win.img.cols.collisionBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.dbg.VCS().Mem.Poke(cpubus.ReadAddress[cpubus.CXM0FB], v)
				})
			})

		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.AlignTextToFramePadding()
		imgui.Text("CXM1FB")
		imgui.TableNextColumn()
		drawRegister("##CXM1FB", cxm1fb, vcs.TIADrivenPins, win.img.cols.collisionBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.dbg.VCS().Mem.Poke(cpubus.ReadAddress[cpubus.CXM1FB], v)
				})
			})

		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.AlignTextToFramePadding()
		imgui.Text("CXBLPF")
		imgui.TableNextColumn()
		drawRegister("##CXBLPF", cxblpf, vcs.TIADrivenPins, win.img.cols.collisionBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.dbg.VCS().Mem.Poke(cpubus.ReadAddress[cpubus.CXBLPF], v)
				})
			})

		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.AlignTextToFramePadding()
		imgui.Text("CXPPMM")
		imgui.TableNextColumn()
		drawRegister("##CXPPMM", cxppmm, vcs.TIADrivenPins, win.img.cols.collisionBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.dbg.VCS().Mem.Poke(cpubus.ReadAddress[cpubus.CXPPMM], v)
				})
			})

		imgui.EndTable()
	}

	imgui.Spacing()

	if imgui.Button("Clear Collisions") {
		win.img.dbg.PushFunction(func() {
			win.img.dbg.VCS().TIA.Video.Collisions.Clear()
		})
	}
}
