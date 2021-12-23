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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
)

const winPortsID = "Ports"

type winPorts struct {
	img  *SdlImgui
	open bool
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

func (win *winPorts) isOpen() bool {
	return win.open
}

func (win *winPorts) setOpen(open bool) {
	win.open = open
}

func (win *winPorts) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{462, 121}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})

	if !imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize) {
		imgui.End()
		return
	}
	defer imgui.End()

	if imgui.BeginTableV("riotSWCHx", 6, imgui.TableFlagsNone, imgui.Vec2{}, 0) {
		// CPU written SWCHx values
		imgui.TableNextRow()
		imgui.TableNextColumn()
		imguiLabel(fmt.Sprintf("%c", fonts.Chip))

		imgui.TableNextColumn()
		imguiLabel("SWCHA")

		imgui.TableNextColumn()
		win.drawRegister("##SWCHA_W", win.img.lz.Ports.SWCHA_W, 0xff, true,
			func(v uint8) {
				win.img.vcs.RIOT.Ports.SetField("swcha_w", v)
			})

		imgui.TableNextColumn()
		imguiLabel(fmt.Sprintf("%c", fonts.Chip))

		imgui.TableNextColumn()
		imguiLabel("SWCHB")

		imgui.TableNextColumn()
		win.drawRegister("##SWCHB_W", win.img.lz.Ports.SWCHB_W, 0xff, true,
			func(v uint8) {
				win.img.vcs.RIOT.Ports.SetField("swchb_w", v)
			})

		// SWCHx CNT flags
		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("SWACNT")

		imgui.TableNextColumn()
		win.drawRegister("##SWACNT", win.img.lz.Ports.SWACNT, 0xff, true,
			func(v uint8) {
				win.img.vcs.RIOT.Ports.SetField("swacnt", v)
			})

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("SWBCNT")

		imgui.TableNextColumn()
		win.drawRegister("##SWBCNT", win.img.lz.Ports.SWBCNT, 0xff, true,
			func(v uint8) {
				win.img.vcs.RIOT.Ports.SetField("swbcnt", v)
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
		win.drawRegister("##SWCHA_R", win.img.lz.Ports.SWCHA, 0xff, true,
			func(v uint8) {
				win.img.vcs.RIOT.Ports.SetField("swcha", v)
			})

		imgui.TableNextColumn()
		if win.img.lz.Ports.SWCHB != win.img.lz.Ports.SWCHB_Derived {
			imguiLabel(fmt.Sprintf("%c", fonts.Unlocked))
		}

		imgui.TableNextColumn()
		imguiLabel("SWCHB")

		imgui.TableNextColumn()
		win.drawRegister("##SWCHB_R", win.img.lz.Ports.SWCHB, 0xff, true,
			func(v uint8) {
				win.img.vcs.RIOT.Ports.SetField("swchb", v)
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
		win.drawRegister("##INPT0", win.img.lz.Ports.INPT0, addresses.DataMasks[addresses.INPT0], false,
			func(v uint8) {
				err := win.img.vcs.Mem.Poke(addresses.ReadAddress["INPT0"], v)
				if err != nil {
					panic(err)
				}
			})

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("INPT1")

		imgui.TableNextColumn()
		win.drawRegister("##INPT1", win.img.lz.Ports.INPT1, addresses.DataMasks[addresses.INPT1], false,
			func(v uint8) {
				err := win.img.vcs.Mem.Poke(addresses.ReadAddress["INPT1"], v)
				if err != nil {
					panic(err)
				}
			})

		imgui.TableNextRow()

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("INPT2")

		imgui.TableNextColumn()
		win.drawRegister("##INPT2", win.img.lz.Ports.INPT2, addresses.DataMasks[addresses.INPT2], false,
			func(v uint8) {
				err := win.img.vcs.Mem.Poke(addresses.ReadAddress["INPT2"], v)
				if err != nil {
					panic(err)
				}
			})

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("INPT3")

		imgui.TableNextColumn()
		win.drawRegister("##INPT3", win.img.lz.Ports.INPT3, addresses.DataMasks[addresses.INPT3], false,
			func(v uint8) {
				err := win.img.vcs.Mem.Poke(addresses.ReadAddress["INPT3"], v)
				if err != nil {
					panic(err)
				}
			})

		imgui.TableNextRow()

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("INPT4")

		imgui.TableNextColumn()
		win.drawRegister("##INPT4", win.img.lz.Ports.INPT4, addresses.DataMasks[addresses.INPT4], false,
			func(v uint8) {
				err := win.img.vcs.Mem.Poke(addresses.ReadAddress["INPT4"], v)
				if err != nil {
					panic(err)
				}
			})

		imgui.TableNextColumn()
		imgui.TableNextColumn()
		imguiLabel("INPT5")

		imgui.TableNextColumn()
		win.drawRegister("##INPT5", win.img.lz.Ports.INPT5, addresses.DataMasks[addresses.INPT5], false,
			func(v uint8) {
				err := win.img.vcs.Mem.Poke(addresses.ReadAddress["INPT5"], v)
				if err != nil {
					panic(err)
				}
			})

		imgui.EndTable()
	}

	// poking chip registers may not have the effect the user
	// expects (compare to poking CPU registers for example)
	// !!TODO: warning/help text for chip registers window
}

func (win *winPorts) drawRegister(id string, val uint8, mask uint8, bits bool, onWrite func(uint8)) {
	v := fmt.Sprintf("%02x", val)
	if imguiHexInput(id, 2, &v) {
		v, err := strconv.ParseUint(v, 16, 8)
		if err != nil {
			panic(err)
		}
		win.img.dbg.PushRawEvent(func() { onWrite(uint8(v)) })
	}

	imgui.SameLine()

	seq := newDrawlistSequence(win.img, imgui.Vec2{X: imgui.FrameHeight() * 0.75, Y: imgui.FrameHeight() * 0.75}, true)
	for i := 0; i < 8; i++ {
		if mask<<i&0x80 == 0x80 {
			if (val<<i)&0x80 != 0x80 {
				seq.nextItemDepressed = true
			}
			if seq.rectFill(win.img.cols.riotIOBit) {
				v := val ^ (0x80 >> i)
				win.img.dbg.PushRawEvent(func() { onWrite(uint8(v)) })
			}
		} else {
			seq.nextItemDepressed = true
			seq.rectEmpty(win.img.cols.riotIOBit)
		}
		seq.sameLine()
	}
	seq.end()
}
