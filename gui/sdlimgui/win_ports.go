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

func (win *winPorts) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 462, Y: 121}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
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
		swcha_w := win.img.cache.VCS.RIOT.Ports.PeekField("swcha_w").(uint8)
		drawRegister("##SWCHA_W", swcha_w, 0xff, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.dbg.VCS().RIOT.Ports.PokeField("swcha_w", v)
				})
			})

		imgui.TableNextColumn()
		imguiLabel(fmt.Sprintf("%c", fonts.Chip))

		imgui.TableNextColumn()
		imguiLabel("SWCHB")

		imgui.TableNextColumn()
		swchb_w := win.img.cache.VCS.RIOT.Ports.PeekField("swchb_w").(uint8)
		drawRegister("##SWCHB_W", swchb_w, 0xff, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.dbg.VCS().RIOT.Ports.PokeField("swchb_w", v)
				})
			})

		// SWCHx CNT flags
		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("SWACNT")

		imgui.TableNextColumn()
		swacnt := win.img.cache.VCS.RIOT.Ports.PeekField("swacnt").(uint8)
		drawRegister("##SWACNT", swacnt, 0xff, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.dbg.VCS().RIOT.Ports.PokeField("swacnt", v)
				})
			})

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("SWBCNT")

		imgui.TableNextColumn()
		swbcnt := win.img.cache.VCS.RIOT.Ports.PeekField("swbcnt").(uint8)
		drawRegister("##SWBCNT", swbcnt, 0xff, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.dbg.VCS().RIOT.Ports.PokeField("swbcnt", v)
				})
			})

		// actual SWCHx values
		imgui.TableNextRow()
		imgui.TableNextColumn()
		swcha := win.img.cache.VCS.RIOT.Ports.PeekField("swcha").(uint8)
		swcha_derived := win.img.cache.VCS.RIOT.Ports.PeekField("swcha_derived").(uint8)
		if swcha != swcha_derived {
			imguiLabel(fmt.Sprintf("%c", fonts.Unlocked))
		}

		imgui.TableNextColumn()
		imguiLabel("SWCHA")

		imgui.TableNextColumn()
		drawRegister("##SWCHA_R", swcha, 0xff, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.dbg.VCS().RIOT.Ports.PokeField("swcha", v)
				})
			})

		imgui.TableNextColumn()
		swchb := win.img.cache.VCS.RIOT.Ports.PeekField("swchb").(uint8)
		swchb_derived := win.img.cache.VCS.RIOT.Ports.PeekField("swchb_derived").(uint8)
		if swchb != swchb_derived {
			imguiLabel(fmt.Sprintf("%c", fonts.Unlocked))
		}

		imgui.TableNextColumn()
		imguiLabel("SWCHB")

		imgui.TableNextColumn()
		drawRegister("##SWCHB_R", swchb, 0xff, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					win.img.dbg.VCS().RIOT.Ports.PokeField("swchb", v)
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
		inpt0, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddressByRegister[cpubus.INPT0])
		drawRegister("##INPT0", inpt0, vcs.TIADrivenPins, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					err := win.img.dbg.VCS().Mem.Poke(cpubus.ReadAddressByRegister[cpubus.INPT0], v)
					if err != nil {
						panic(err)
					}
				})
			})

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("INPT1")

		imgui.TableNextColumn()
		inpt1, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddressByRegister[cpubus.INPT1])
		drawRegister("##INPT1", inpt1, vcs.TIADrivenPins, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					err := win.img.dbg.VCS().Mem.Poke(cpubus.ReadAddressByRegister[cpubus.INPT1], v)
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
		inpt2, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddressByRegister[cpubus.INPT2])
		drawRegister("##INPT2", inpt2, vcs.TIADrivenPins, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					err := win.img.dbg.VCS().Mem.Poke(cpubus.ReadAddressByRegister[cpubus.INPT2], v)
					if err != nil {
						panic(err)
					}
				})
			})

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("INPT3")

		imgui.TableNextColumn()
		inpt3, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddressByRegister[cpubus.INPT3])
		drawRegister("##INPT3", inpt3, vcs.TIADrivenPins, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					err := win.img.dbg.VCS().Mem.Poke(cpubus.ReadAddressByRegister[cpubus.INPT3], v)
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
		inpt4, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddressByRegister[cpubus.INPT4])
		drawRegister("##INPT4", inpt4, vcs.TIADrivenPins, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					err := win.img.dbg.VCS().Mem.Poke(cpubus.ReadAddressByRegister[cpubus.INPT4], v)
					if err != nil {
						panic(err)
					}
				})
			})

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("INPT5")

		imgui.TableNextColumn()
		inpt5, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddressByRegister[cpubus.INPT5])
		drawRegister("##INPT5", inpt5, vcs.TIADrivenPins, win.img.cols.portsBit,
			func(v uint8) {
				win.img.dbg.PushFunction(func() {
					err := win.img.dbg.VCS().Mem.Poke(cpubus.ReadAddressByRegister[cpubus.INPT5], v)
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
