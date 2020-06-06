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

	"github.com/inkyblackness/imgui-go/v2"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
)

const winDPCregistersTitle = "DPC Registers"

type winDPCregisters struct {
	windowManagement
	widgetDimensions

	img *SdlImgui
}

func newWinDPCregisters(img *SdlImgui) (managedWindow, error) {
	win := &winDPCregisters{
		img: img,
	}

	return win, nil
}

func (win *winDPCregisters) init() {
	win.widgetDimensions.init()
}

func (win *winDPCregisters) destroy() {
}

func (win *winDPCregisters) id() string {
	return winDPCregistersTitle
}

func (win *winDPCregisters) draw() {
	if !win.open {
		return
	}

	// do not open window if there is no valid cartridge debug bus available
	r, ok := win.img.lz.Cart.Registers.(cartridge.DPCregisters)
	if !win.img.lz.Cart.HasDebugBus || !ok {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{659, 35}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winDPCregistersTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	// random number generator value
	rng := fmt.Sprintf("%02x", r.RNG)
	imguiText("Random Number Generator")
	imgui.PushItemWidth(win.twoDigitDim.X)
	if imguiHexInput("##rng", !win.img.paused, 2, &rng) {
		win.img.lz.Dbg.PushRawEvent(func() {
			b := win.img.lz.Dbg.VCS.Mem.Cart.GetDebugBus()
			b.PutRegister("rng", rng)
		})
	}
	imgui.PopItemWidth()

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	// loop over data fetchers
	imgui.Text("Data Fetchers")
	imgui.Spacing()
	for i := 0; i < len(r.Fetcher); i++ {
		f := i

		imguiText(fmt.Sprintf("#%d", f))

		label := fmt.Sprintf("##%dlow", i)
		low := fmt.Sprintf("%02x", r.Fetcher[i].Low)
		imguiText("Low")
		imgui.PushItemWidth(win.twoDigitDim.X)
		if imguiHexInput(label, !win.img.paused, 2, &low) {
			win.img.lz.Dbg.PushRawEvent(func() {
				b := win.img.lz.Dbg.VCS.Mem.Cart.GetDebugBus()
				b.PutRegister(fmt.Sprintf("fetcher::%d::low", f), low)
			})
		}
		imgui.PopItemWidth()

		imgui.SameLine()
		label = fmt.Sprintf("##%dhi", i)
		hi := fmt.Sprintf("%02x", r.Fetcher[i].Hi)
		imguiText("Hi")
		imgui.PushItemWidth(win.twoDigitDim.X)
		if imguiHexInput(label, !win.img.paused, 2, &hi) {
			win.img.lz.Dbg.PushRawEvent(func() {
				b := win.img.lz.Dbg.VCS.Mem.Cart.GetDebugBus()
				b.PutRegister(fmt.Sprintf("fetcher::%d::hi", f), hi)
			})
		}
		imgui.PopItemWidth()

		imgui.SameLine()
		label = fmt.Sprintf("##%dtop", i)
		top := fmt.Sprintf("%02x", r.Fetcher[i].Top)
		imguiText("Top")
		imgui.PushItemWidth(win.twoDigitDim.X)
		if imguiHexInput(label, !win.img.paused, 2, &top) {
			win.img.lz.Dbg.PushRawEvent(func() {
				b := win.img.lz.Dbg.VCS.Mem.Cart.GetDebugBus()
				b.PutRegister(fmt.Sprintf("fetcher::%d::top", f), top)
			})
		}
		imgui.PopItemWidth()

		imgui.SameLine()
		label = fmt.Sprintf("##%dbottom", i)
		bottom := fmt.Sprintf("%02x", r.Fetcher[i].Bottom)
		imguiText("Bottom")
		imgui.PushItemWidth(win.twoDigitDim.X)
		if imguiHexInput(label, !win.img.paused, 2, &bottom) {
			win.img.lz.Dbg.PushRawEvent(func() {
				b := win.img.lz.Dbg.VCS.Mem.Cart.GetDebugBus()
				b.PutRegister(fmt.Sprintf("fetcher::%d::bottom", f), bottom)
			})
		}
		imgui.PopItemWidth()

		// data fetchers 4-7 can be set to "music mode"
		if i >= 4 {
			imgui.SameLine()
			mm := r.Fetcher[i].MusicMode
			if imgui.Checkbox(fmt.Sprintf("##%dmusicmode", i), &mm) {
				win.img.lz.Dbg.PushRawEvent(func() {
					b := win.img.lz.Dbg.VCS.Mem.Cart.GetDebugBus()
					b.PutRegister(fmt.Sprintf("fetcher::%d::musicmode", f), fmt.Sprintf("%v", mm))
				})
			}
		}
	}

	imgui.End()
}
