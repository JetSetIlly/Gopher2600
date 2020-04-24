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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlimgui

import (
	"fmt"
	"strconv"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"

	"github.com/inkyblackness/imgui-go/v2"
)

const winRAMTitle = "RAM"

type winRAM struct {
	windowManagement
	img *SdlImgui

	// for convenience we represent internal VCS RAM using the RAMinfo struct
	// from the cartridge package. this facilitates the drawGrid() function
	// below.
	vcsRAMinfo cartridge.RAMinfo

	// widget dimensions
	byteDim imgui.Vec2

	// we know this value after the first pass
	headerRowStart float32

	// height required to display VCS RAM in its entirity. calculated value
	vcsRAMheight float32
}

func newWinRAM(img *SdlImgui) (managedWindow, error) {
	win := &winRAM{
		img: img,
		vcsRAMinfo: cartridge.RAMinfo{
			Label:       "VCS",
			ReadOrigin:  0x80,
			ReadMemtop:  0xff,
			WriteOrigin: 0x80,
			WriteMemtop: 0xff,
		},
	}

	return win, nil
}

func (win *winRAM) init() {
	win.byteDim = imguiGetFrameDim("FF")
}

func (win *winRAM) destroy() {
}

func (win *winRAM) id() string {
	return winRAMTitle
}

// draw is called by service loop
func (win *winRAM) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{890, 29}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winRAMTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	if len(win.img.lazy.Cart.RAMinfo) > 0 {
		imgui.BeginTabBar("")

		if imgui.BeginTabItemV(win.vcsRAMinfo.Label, nil, 0) {

			// calculate the height required to display VCS RAM in its
			// entirity. we reuse this to limit the amount of space used to
			// show cart RAM
			vcsRAMheight := imgui.CursorPosY()
			win.drawGrid(win.vcsRAMinfo)
			win.vcsRAMheight = imgui.CursorPosY() - vcsRAMheight

			imgui.EndTabItem()
		}

		for i := 0; i < len(win.img.lazy.Cart.RAMinfo); i++ {
			if win.img.lazy.Cart.RAMinfo[i].Active {
				if imgui.BeginTabItemV(win.img.lazy.Cart.RAMinfo[i].Label, nil, 0) {

					// display cart RAM and limit the amount of space it requires
					imgui.BeginChildV(fmt.Sprintf("cartRAM##%d", i), imgui.Vec2{X: 0, Y: win.vcsRAMheight}, false, 0)
					win.drawGrid(win.img.lazy.Cart.RAMinfo[i])
					imgui.EndChild()

					imgui.EndTabItem()
				}
			}
		}
		imgui.EndTabBar()
	} else {
		win.drawGrid(win.vcsRAMinfo)
	}

	imgui.End()
}

func (win *winRAM) drawGrid(raminfo cartridge.RAMinfo) {
	// no spacing between any of the items in the RAM window
	imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, imgui.Vec2{})

	// draw headers for each column. this relies on headerRowStart, which requires 1
	// frame to decide before it is accurate.
	headerDim := imgui.Vec2{X: win.headerRowStart, Y: imgui.CursorPosY()}
	for i := 0; i < 16; i++ {
		imgui.SetCursorPos(headerDim)
		headerDim.X += win.byteDim.X
		imgui.AlignTextToFramePadding()
		imgui.Text(fmt.Sprintf("-%x", i))
	}

	// draw rows
	imgui.PushItemWidth(win.byteDim.X)
	i := uint16(0)
	for readAddr := raminfo.ReadOrigin; readAddr <= raminfo.ReadMemtop; readAddr++ {
		// draw row header
		if i%16 == 0 {
			imgui.AlignTextToFramePadding()
			imgui.Text(fmt.Sprintf("%02x- ", readAddr/16))
			imgui.SameLine()
			win.headerRowStart = imgui.CursorPosX()
		} else {
			imgui.SameLine()
		}
		win.drawEditByte(raminfo, readAddr, raminfo.WriteOrigin+i)
		i++
	}
	imgui.PopItemWidth()

	// finished with spacing setting
	imgui.PopStyleVar()
}

func (win *winRAM) drawEditByte(raminfo cartridge.RAMinfo, readAddr uint16, writeAddr uint16) {
	d := win.img.lazy.ReadRAM(raminfo, readAddr)

	label := fmt.Sprintf("##%d", readAddr)
	content := fmt.Sprintf("%02x", d)

	if imguiHexInput(label, !win.img.paused, 2, &content) {
		if v, err := strconv.ParseUint(content, 16, 8); err == nil {
			// we don't know if this address is from the internal RAM or from
			// an area of cartridge RAM. for this reason we're sending the
			// write through the high-level memory write, which will map the
			// address for us.
			win.img.lazy.Dbg.PushRawEvent(func() {
				win.img.lazy.VCS.Mem.Write(writeAddr, uint8(v))
			})
		}
	}
}
