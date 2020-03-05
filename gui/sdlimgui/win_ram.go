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
	"gopher2600/hardware/memory/cartridge"
	"strconv"

	"github.com/inkyblackness/imgui-go/v2"
)

const winRAMTitle = "RAM"

type winRAM struct {
	windowManagement
	img *SdlImgui

	// for convenience we represent internal VCS RAM using the RAMinfo struct
	// from the cartridge package. this facilitates the drawGrid() function
	// below.
	vcsRAM cartridge.RAMinfo

	// widget dimensions
	byteDim imgui.Vec2

	// we know this value after the first pass
	headerRowStart float32
}

func newWinRAM(img *SdlImgui) (managedWindow, error) {
	win := &winRAM{
		img: img,
		vcsRAM: cartridge.RAMinfo{
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

	imgui.SetNextWindowPosV(imgui.Vec2{883, 35}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winRAMTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	cartRAM := win.img.vcs.Mem.Cart.GetRAMinfo()

	if cartRAM != nil {
		imgui.BeginTabBar("")
		if imgui.BeginTabItemV(win.vcsRAM.Label, nil, 0) {
			win.drawGrid(win.vcsRAM)
			imgui.EndTabItem()
		}
		for i := 0; i < len(cartRAM); i++ {
			if cartRAM[i].Active {
				if imgui.BeginTabItemV(cartRAM[i].Label, nil, 0) {
					win.drawGrid(cartRAM[i])
					imgui.EndTabItem()
				}
			}
		}
		imgui.EndTabBar()
	} else {
		// there is no cartrdige memory so we don't need a Tab bar, we'll just
		// draw the RAM grid
		win.drawGrid(win.vcsRAM)
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
	j := uint16(0)
	for i := raminfo.ReadOrigin; i <= raminfo.ReadMemtop; i++ {
		// draw row header
		if j%16 == 0 {
			imgui.AlignTextToFramePadding()
			imgui.Text(fmt.Sprintf("%02x- ", i/16))
			imgui.SameLine()
			win.headerRowStart = imgui.CursorPosX()
		} else {
			imgui.SameLine()
		}
		win.drawEditByte(i, raminfo.WriteOrigin+j)
		j++
	}
	imgui.PopItemWidth()

	// finished with spacing setting
	imgui.PopStyleVar()
}

func (win *winRAM) drawEditByte(readAddr uint16, writeAddr uint16) {
	label := fmt.Sprintf("##%d", readAddr)
	d, _ := win.img.vcs.Mem.Read(readAddr)
	content := fmt.Sprintf("%02x", d)

	if imguiHexInput(label, !win.img.paused, 2, &content) {
		if v, err := strconv.ParseUint(content, 16, 8); err == nil {
			// we don't know if this address is from the internal RAM or from
			// an area of cartridge RAM. for this reason we're sending the
			// write through the high-level memory write, which will map the
			// address for us.
			win.img.vcs.Mem.Write(writeAddr, uint8(v))
		}
	}
}
