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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/memory/vcs"
)

const winPortsID = "Ports"

type winPorts struct {
	debuggerWin

	img *SdlImgui
}

func newWinPorts(img *SdlImgui) (window, error) {
	win := &winPorts{
		img: img,
	}

	return win, nil
}

func (win *winPorts) init() {
}

func (win *winPorts) id() string {
	return winPortsID
}

func (win *winPorts) debuggerDraw() {
	if !win.debuggerOpen {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{462, 121}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()
}

func (win *winPorts) draw() {
	if imgui.BeginTableV("riotSWCHx", 6, imgui.TableFlagsNone, imgui.Vec2{}, 0) {
		// CPU written SWCHx values
		imgui.TableNextRow()
		imgui.TableNextColumn()
		imguiLabel(fmt.Sprintf("%c", fonts.Chip))

		imgui.TableNextColumn()
		imguiLabel("SWCHA")

		imgui.TableNextColumn()
		drawRegister("##SWCHA_W", win.img.lz.Ports.SWCHA_W, 0xff, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.vcs.RIOT.Ports.PokeField("swcha_w", v)
				})
			})

		imgui.TableNextColumn()
		imguiLabel(fmt.Sprintf("%c", fonts.Chip))

		imgui.TableNextColumn()
		imguiLabel("SWCHB")

		imgui.TableNextColumn()
		drawRegister("##SWCHB_W", win.img.lz.Ports.SWCHB_W, 0xff, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.vcs.RIOT.Ports.PokeField("swchb_w", v)
				})
			})

		// SWCHx CNT flags
		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("SWACNT")

		imgui.TableNextColumn()
		drawRegister("##SWACNT", win.img.lz.Ports.SWACNT, 0xff, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.vcs.RIOT.Ports.PokeField("swacnt", v)
				})
			})

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("SWBCNT")

		imgui.TableNextColumn()
		drawRegister("##SWBCNT", win.img.lz.Ports.SWBCNT, 0xff, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.vcs.RIOT.Ports.PokeField("swbcnt", v)
				})
			})

		// actual SWCHx values
		imgui.TableNextRow()
		imgui.TableNextColumn()
		if win.img.lz.Ports.SWCHA != win.img.lz.Ports.SWCHA_Derived {
			imguiLabel(fmt.Sprintf("%c", fonts.Unlocked))
		}

		imgui.TableNextColumn()
		imguiLabel("SWCHA")

		imgui.TableNextColumn()
		drawRegister("##SWCHA_R", win.img.lz.Ports.SWCHA, 0xff, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.vcs.RIOT.Ports.PokeField("swcha", v)
				})
			})

		imgui.TableNextColumn()
		if win.img.lz.Ports.SWCHB != win.img.lz.Ports.SWCHB_Derived {
			imguiLabel(fmt.Sprintf("%c", fonts.Unlocked))
		}

		imgui.TableNextColumn()
		imguiLabel("SWCHB")

		imgui.TableNextColumn()
		drawRegister("##SWCHB_R", win.img.lz.Ports.SWCHB, 0xff, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.vcs.RIOT.Ports.PokeField("swchb", v)
				})
			})

		imgui.EndTable()
	}

	imgui.Separator()

	if imgui.BeginTableV("riotINPTx", 6, imgui.TableFlagsSizingStretchProp, imgui.Vec2{}, 0) {
		imgui.TableNextRow()

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("INPT0")

		imgui.TableNextColumn()
		drawRegister("##INPT0", win.img.lz.Ports.INPT0, vcs.TIADrivenPins, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					err := win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.INPT0], v)
					if err != nil {
						panic(err)
					}
				})
			})

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("INPT1")

		imgui.TableNextColumn()
		drawRegister("##INPT1", win.img.lz.Ports.INPT1, vcs.TIADrivenPins, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					err := win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.INPT1], v)
					if err != nil {
						panic(err)
					}
				})
			})

		imgui.TableNextRow()

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("INPT2")

		imgui.TableNextColumn()
		drawRegister("##INPT2", win.img.lz.Ports.INPT2, vcs.TIADrivenPins, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					err := win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.INPT2], v)
					if err != nil {
						panic(err)
					}
				})
			})

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("INPT3")

		imgui.TableNextColumn()
		drawRegister("##INPT3", win.img.lz.Ports.INPT3, vcs.TIADrivenPins, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					err := win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.INPT3], v)
					if err != nil {
						panic(err)
					}
				})
			})

		imgui.TableNextRow()

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("INPT4")

		imgui.TableNextColumn()
		drawRegister("##INPT4", win.img.lz.Ports.INPT4, vcs.TIADrivenPins, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					err := win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.INPT4], v)
					if err != nil {
						panic(err)
					}
				})
			})

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("INPT5")

		imgui.TableNextColumn()
		drawRegister("##INPT5", win.img.lz.Ports.INPT5, vcs.TIADrivenPins, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					err := win.img.vcs.Mem.Poke(cpubus.ReadAddress[cpubus.INPT5], v)
					if err != nil {
						panic(err)
					}
				})
			})

		imgui.EndTable()
	}

	// poking chip registers may not have the effect the user
	// expects (compare to poking CPU registers for example)
	// !!TODO: warning/help text for chip registers window
}
