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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
)

const winDPCregistersID = "DPC Registers"

type winDPCregisters struct {
	debuggerWin

	img *SdlImgui
}

func newWinDPCregisters(img *SdlImgui) (window, error) {
	win := &winDPCregisters{
		img: img,
	}

	return win, nil
}

func (win *winDPCregisters) init() {
}

func (win *winDPCregisters) id() string {
	return winDPCregistersID
}

func (win *winDPCregisters) debuggerDraw() {
	if !win.debuggerOpen {
		return
	}

	// do not open window if there is no valid cartridge debug bus available
	_, ok := win.img.lz.Cart.Registers.(cartridge.DPCregisters)
	if !win.img.lz.Cart.HasRegistersBus || !ok {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{255, 153}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()
}

func (win *winDPCregisters) draw() {
	r := win.img.lz.Cart.Registers.(cartridge.DPCregisters)

	// random number generator value
	rng := fmt.Sprintf("%02x", r.RNG)
	imguiLabel("Random Number Generator")
	if imguiHexInput("##rng", 2, &rng) {
		win.img.dbg.PushFunction(func() {
			b := win.img.vcs.Mem.Cart.GetRegistersBus()
			b.PutRegister("rng", rng)
		})
	}

	imguiSeparator()

	// loop over data fetchers
	imgui.Text("Data Fetchers")
	imgui.Spacing()
	for i := 0; i < len(r.Fetcher); i++ {
		f := i

		imguiLabel(fmt.Sprintf("%d.", f))

		label := fmt.Sprintf("##%dlow", i)
		low := fmt.Sprintf("%02x", r.Fetcher[i].Low)
		imguiLabel("Low")
		if imguiHexInput(label, 2, &low) {
			win.img.dbg.PushFunction(func() {
				b := win.img.vcs.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("datafetcher::%d::low", f), low)
			})
		}

		imgui.SameLine()
		label = fmt.Sprintf("##%dhi", i)
		hi := fmt.Sprintf("%02x", r.Fetcher[i].Hi)
		imguiLabel("Hi")
		if imguiHexInput(label, 2, &hi) {
			win.img.dbg.PushFunction(func() {
				b := win.img.vcs.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("datafetcher::%d::hi", f), hi)
			})
		}

		imgui.SameLine()
		label = fmt.Sprintf("##%dtop", i)
		top := fmt.Sprintf("%02x", r.Fetcher[i].Top)
		imguiLabel("Top")
		if imguiHexInput(label, 2, &top) {
			win.img.dbg.PushFunction(func() {
				b := win.img.vcs.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("datafetcher::%d::top", f), top)
			})
		}

		imgui.SameLine()
		label = fmt.Sprintf("##%dbottom", i)
		bottom := fmt.Sprintf("%02x", r.Fetcher[i].Bottom)
		imguiLabel("Bottom")
		if imguiHexInput(label, 2, &bottom) {
			win.img.dbg.PushFunction(func() {
				b := win.img.vcs.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("datafetcher::%d::bottom", f), bottom)
			})
		}

		// data fetchers 4-7 can be set to "music mode"
		if i >= 4 {
			imgui.SameLine()
			mm := r.Fetcher[i].MusicMode
			if imgui.Checkbox(fmt.Sprintf("##%dmusicmode", i), &mm) {
				win.img.dbg.PushFunction(func() {
					b := win.img.vcs.Mem.Cart.GetRegistersBus()
					b.PutRegister(fmt.Sprintf("datafetcher::%d::musicmode", f), fmt.Sprintf("%v", mm))
				})
			}
		}
	}
}
