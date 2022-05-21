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

func (win *winCollisions) debuggerDraw() {
	if !win.debuggerOpen {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{530, 455}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()
}

func (win *winCollisions) draw() {
	if imgui.BeginTableV("##collisions", 2, imgui.TableFlagsNone, imgui.Vec2{}, 0.0) {

		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.AlignTextToFramePadding()
		imgui.Text("CXM0P ")
		imgui.TableNextColumn()
		drawRegister("##CXM0P", win.img.lz.Collisions.CXM0P, vcs.TIADrivenPins, win.img.cols.collisionBit,
			func(v uint8) {
				win.img.dbg.PushRawEvent(func() {
					win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.CXM0P], v)
				})
			})

		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.AlignTextToFramePadding()
		imgui.Text("CXM1P ")
		imgui.TableNextColumn()
		drawRegister("##CXM1P", win.img.lz.Collisions.CXM1P, vcs.TIADrivenPins, win.img.cols.collisionBit,
			func(v uint8) {
				win.img.dbg.PushRawEvent(func() {
					win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.CXM1P], v)
				})
			})

		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.AlignTextToFramePadding()
		imgui.Text("CXP0FB")
		imgui.TableNextColumn()
		drawRegister("##CXP0FB", win.img.lz.Collisions.CXP0FB, vcs.TIADrivenPins, win.img.cols.collisionBit,
			func(v uint8) {
				win.img.dbg.PushRawEvent(func() {
					win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.CXP0FB], v)
				})
			})

		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.AlignTextToFramePadding()
		imgui.Text("CXP1FB")
		imgui.TableNextColumn()
		drawRegister("##CXP1FB", win.img.lz.Collisions.CXP1FB, vcs.TIADrivenPins, win.img.cols.collisionBit,
			func(v uint8) {
				win.img.dbg.PushRawEvent(func() {
					win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.CXP1FB], v)
				})
			})

		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.AlignTextToFramePadding()
		imgui.Text("CXM0FB")
		imgui.TableNextColumn()
		drawRegister("##CXM0FB", win.img.lz.Collisions.CXM0FB, vcs.TIADrivenPins, win.img.cols.collisionBit,
			func(v uint8) {
				win.img.dbg.PushRawEvent(func() {
					win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.CXM0FB], v)
				})
			})

		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.AlignTextToFramePadding()
		imgui.Text("CXM1FB")
		imgui.TableNextColumn()
		drawRegister("##CXM1FB", win.img.lz.Collisions.CXM1FB, vcs.TIADrivenPins, win.img.cols.collisionBit,
			func(v uint8) {
				win.img.dbg.PushRawEvent(func() {
					win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.CXM1FB], v)
				})
			})

		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.AlignTextToFramePadding()
		imgui.Text("CXBLPF")
		imgui.TableNextColumn()
		drawRegister("##CXBLPF", win.img.lz.Collisions.CXBLPF, vcs.TIADrivenPins, win.img.cols.collisionBit,
			func(v uint8) {
				win.img.dbg.PushRawEvent(func() {
					win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.CXBLPF], v)
				})
			})

		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.AlignTextToFramePadding()
		imgui.Text("CXPPMM")
		imgui.TableNextColumn()
		drawRegister("##CXPPMM", win.img.lz.Collisions.CXPPMM, vcs.TIADrivenPins, win.img.cols.collisionBit,
			func(v uint8) {
				win.img.dbg.PushRawEvent(func() {
					win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.CXPPMM], v)
				})
			})

		imgui.EndTable()
	}

	imgui.Spacing()

	if imgui.Button("Clear Collisions") {
		win.img.dbg.PushRawEvent(func() {
			win.img.vcs.TIA.Video.Collisions.Clear()
		})
	}
}
