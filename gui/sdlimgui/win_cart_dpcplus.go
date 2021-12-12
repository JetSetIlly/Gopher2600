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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/dpcplus"
)

const winDPCplusRegistersID = "DPC+ Registers"

type winDPCplusRegisters struct {
	img  *SdlImgui
	open bool
}

func newWinDPCplusRegisters(img *SdlImgui) (window, error) {
	win := &winDPCplusRegisters{
		img: img,
	}

	return win, nil
}

func (win *winDPCplusRegisters) init() {
}

func (win *winDPCplusRegisters) id() string {
	return winDPCplusRegistersID
}

func (win *winDPCplusRegisters) isOpen() bool {
	return win.open
}

func (win *winDPCplusRegisters) setOpen(open bool) {
	win.open = open
}

func (win *winDPCplusRegisters) draw() {
	if !win.open {
		return
	}

	// do not open window if there is no valid cartridge debug bus available
	r, ok := win.img.lz.Cart.Registers.(dpcplus.Registers)
	if !win.img.lz.Cart.HasRegistersBus || !ok {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{256, 192}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)

	// random number generator value
	rng := fmt.Sprintf("%08x", r.RNG.Value)
	imguiLabel("Random Number Generator")
	if imguiHexInput("##rng", 8, &rng) {
		win.img.dbg.PushRawEvent(func() {
			b := win.img.vcs.Mem.Cart.GetRegistersBus()
			b.PutRegister("rng", rng)
		})
	}

	imgui.SameLineV(0, 20)
	imguiLabel("Fast Fetch")
	ff := r.FastFetch
	if imgui.Checkbox("##fastfetch", &ff) {
		win.img.dbg.PushRawEvent(func() {
			b := win.img.vcs.Mem.Cart.GetRegistersBus()
			b.PutRegister("fastfetch", fmt.Sprintf("%v", ff))
		})
	}

	imguiSeparator()

	// *** data fetchers grouping ***
	imgui.BeginGroup()

	// loop over data fetchers
	imgui.Text("Data Fetchers")
	imgui.Spacing()
	for i := 0; i < len(r.Fetcher); i++ {
		f := i

		imguiLabel(fmt.Sprintf("#%d", f))

		label := fmt.Sprintf("##d%dlow", i)
		low := fmt.Sprintf("%02x", r.Fetcher[i].Low)
		imguiLabel("Low")
		if imguiHexInput(label, 2, &low) {
			win.img.dbg.PushRawEvent(func() {
				b := win.img.vcs.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("datafetcher::%d::low", f), low)
			})
		}

		imgui.SameLine()
		label = fmt.Sprintf("##d%dhi", i)
		hi := fmt.Sprintf("%02x", r.Fetcher[i].Hi)
		imguiLabel("Hi")
		if imguiHexInput(label, 2, &hi) {
			win.img.dbg.PushRawEvent(func() {
				b := win.img.vcs.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("datafetcher::%d::hi", f), hi)
			})
		}

		imgui.SameLine()
		label = fmt.Sprintf("##d%dtop", i)
		top := fmt.Sprintf("%02x", r.Fetcher[i].Top)
		imguiLabel("Top")
		if imguiHexInput(label, 2, &top) {
			win.img.dbg.PushRawEvent(func() {
				b := win.img.vcs.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("datafetcher::%d::top", f), top)
			})
		}

		imgui.SameLine()
		label = fmt.Sprintf("##d%dbottom", i)
		bottom := fmt.Sprintf("%02x", r.Fetcher[i].Bottom)
		imguiLabel("Bottom")
		if imguiHexInput(label, 2, &bottom) {
			win.img.dbg.PushRawEvent(func() {
				b := win.img.vcs.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("datafetcher::%d::bottom", f), bottom)
			})
		}
	}
	imgui.EndGroup()

	// *** fraction fetchers grouping ***
	imgui.SameLineV(0, 20)
	imgui.BeginGroup()

	// loop over fractional fetchers
	imgui.Text("Fractional Fetchers")
	imgui.Spacing()
	for i := 0; i < len(r.FracFetcher); i++ {
		f := i

		imguiLabel(fmt.Sprintf("#%d", f))

		label := fmt.Sprintf("##f%dlow", i)
		low := fmt.Sprintf("%02x", r.FracFetcher[i].Low)
		imguiLabel("Low")
		if imguiHexInput(label, 2, &low) {
			win.img.dbg.PushRawEvent(func() {
				b := win.img.vcs.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("fractional::%d::low", f), low)
			})
		}

		imgui.SameLine()
		label = fmt.Sprintf("##f%dhi", i)
		hi := fmt.Sprintf("%02x", r.FracFetcher[i].Hi)
		imguiLabel("Hi")
		if imguiHexInput(label, 2, &hi) {
			win.img.dbg.PushRawEvent(func() {
				b := win.img.vcs.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("fractional::%d::hi", f), hi)
			})
		}

		imgui.SameLine()
		label = fmt.Sprintf("##f%dincrement", i)
		increment := fmt.Sprintf("%02x", r.FracFetcher[i].Increment)
		imguiLabel("Increment")
		if imguiHexInput(label, 2, &increment) {
			win.img.dbg.PushRawEvent(func() {
				b := win.img.vcs.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("fractional::%d::increment", f), increment)
			})
		}

		imgui.SameLine()
		label = fmt.Sprintf("##f%dcount", i)
		count := fmt.Sprintf("%02x", r.FracFetcher[i].Count)
		imguiLabel("Count")
		if imguiHexInput(label, 2, &count) {
			win.img.dbg.PushRawEvent(func() {
				b := win.img.vcs.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("fractional::%d::count", f), count)
			})
		}
	}
	imgui.EndGroup()

	// *** music fetchers grouping ***
	imguiSeparator()

	imgui.BeginGroup()

	// loop over music fetchers
	imgui.Text("Music Fetchers")
	imgui.Spacing()
	for i := 0; i < len(r.MusicFetcher); i++ {
		f := i

		imguiLabel(fmt.Sprintf("#%d", f))

		label := fmt.Sprintf("##m%dwaveform", i)
		waveform := fmt.Sprintf("%08x", r.MusicFetcher[i].Waveform)
		imguiLabel("Waveform")
		if imguiHexInput(label, 8, &waveform) {
			win.img.dbg.PushRawEvent(func() {
				b := win.img.vcs.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("music::%d::waveform", f), waveform)
			})
		}

		imgui.SameLine()
		label = fmt.Sprintf("##m%dfeq", i)
		freq := fmt.Sprintf("%08x", r.MusicFetcher[i].Freq)
		imguiLabel("Freq")
		if imguiHexInput(label, 8, &freq) {
			win.img.dbg.PushRawEvent(func() {
				b := win.img.vcs.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("music::%d::freq", f), freq)
			})
		}

		imgui.SameLine()
		label = fmt.Sprintf("##m%dcount", i)
		count := fmt.Sprintf("%08x", r.MusicFetcher[i].Count)
		imguiLabel("Count")
		if imguiHexInput(label, 8, &count) {
			win.img.dbg.PushRawEvent(func() {
				b := win.img.vcs.Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("music::%d::count", f), count)
			})
		}
	}
	imgui.EndGroup()

	imgui.End()
}
