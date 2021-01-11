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

	"github.com/inkyblackness/imgui-go/v3"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
)

const winDPCregistersTitle = "DPC Registers"

type winDPCregisters struct {
	img  *SdlImgui
	open bool
}

func newWinDPCregisters(img *SdlImgui) (window, error) {
	win := &winDPCregisters{
		img: img,
	}

	return win, nil
}

func (win *winDPCregisters) init() {
}

func (win *winDPCregisters) destroy() {
}

func (win *winDPCregisters) id() string {
	return winDPCregistersTitle
}

func (win *winDPCregisters) menuLabel() string {
	return winDPCregistersTitle
}

func (win *winDPCregisters) isOpen() bool {
	return win.open
}

func (win *winDPCregisters) setOpen(open bool) {
	win.open = open
}

func (win *winDPCregisters) draw() {
	if !win.open {
		return
	}

	// do not open window if there is no valid cartridge debug bus available
	r, ok := win.img.lz.Cart.Registers.(cartridge.DPCregisters)
	if !win.img.lz.Cart.HasRegistersBus || !ok {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{633, 451}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winDPCregistersTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	// random number generator value
	rng := fmt.Sprintf("%02x", r.RNG)
	imguiLabel("Random Number Generator")
	if imguiHexInput("##rng", 2, &rng) {
		win.img.lz.Dbg.PushRawEvent(func() {
			b := win.img.lz.Dbg.VCS.Mem.Cart.GetRegistersBus()
			b.PutRegister("rng", rng)
		})
	}

	imguiSeparator()

	// loop over data fetchers
	imgui.Text("Data Fetchers")
	imgui.Spacing()
	for i := 0; i < len(r.Fetcher); i++ {
		f := i

		imguiLabel(fmt.Sprintf("#%d", f))

		label := fmt.Sprintf("##%dlow", i)
		low := fmt.Sprintf("%02x", r.Fetcher[i].Low)
		imguiLabel("Low")
		if imguiHexInput(label, 2, &low) {
			win.img.lz.Dbg.PushRawEvent(func() {
				b := win.img.lz.Dbg.VCS.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("datafetcher::%d::low", f), low)
			})
		}

		imgui.SameLine()
		label = fmt.Sprintf("##%dhi", i)
		hi := fmt.Sprintf("%02x", r.Fetcher[i].Hi)
		imguiLabel("Hi")
		if imguiHexInput(label, 2, &hi) {
			win.img.lz.Dbg.PushRawEvent(func() {
				b := win.img.lz.Dbg.VCS.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("datafetcher::%d::hi", f), hi)
			})
		}

		imgui.SameLine()
		label = fmt.Sprintf("##%dtop", i)
		top := fmt.Sprintf("%02x", r.Fetcher[i].Top)
		imguiLabel("Top")
		if imguiHexInput(label, 2, &top) {
			win.img.lz.Dbg.PushRawEvent(func() {
				b := win.img.lz.Dbg.VCS.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("datafetcher::%d::top", f), top)
			})
		}

		imgui.SameLine()
		label = fmt.Sprintf("##%dbottom", i)
		bottom := fmt.Sprintf("%02x", r.Fetcher[i].Bottom)
		imguiLabel("Bottom")
		if imguiHexInput(label, 2, &bottom) {
			win.img.lz.Dbg.PushRawEvent(func() {
				b := win.img.lz.Dbg.VCS.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("datafetcher::%d::bottom", f), bottom)
			})
		}

		// data fetchers 4-7 can be set to "music mode"
		if i >= 4 {
			imgui.SameLine()
			mm := r.Fetcher[i].MusicMode
			if imgui.Checkbox(fmt.Sprintf("##%dmusicmode", i), &mm) {
				win.img.lz.Dbg.PushRawEvent(func() {
					b := win.img.lz.Dbg.VCS.Mem.Cart.GetRegistersBus()
					b.PutRegister(fmt.Sprintf("datafetcher::%d::musicmode", f), fmt.Sprintf("%v", mm))
				})
			}
		}
	}

	imgui.End()
}
