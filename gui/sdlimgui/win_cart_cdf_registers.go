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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/cdf"
)

const winCDFRegistersID = "CDF Registers"

type winCDFRegisters struct {
	debuggerWin

	img *SdlImgui
}

func newWinCDFRegisters(img *SdlImgui) (window, error) {
	win := &winCDFRegisters{
		img: img,
	}

	return win, nil
}

func (win *winCDFRegisters) init() {
}

func (win *winCDFRegisters) id() string {
	return winCDFRegistersID
}

func (win *winCDFRegisters) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	// do not open window if there is no cartridge registers bus available
	bus := win.img.cache.VCS.Mem.Cart.GetRegistersBus()
	if bus == nil {
		return false
	}
	regs, ok := bus.GetRegisters().(cdf.Registers)
	if !ok {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 610, Y: 303}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw(regs)
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCDFRegisters) draw(regs cdf.Registers) {
	imgui.Text("Datastream")
	imgui.Spacing()

	imgui.BeginGroup()
	imgui.Text("Pointers")
	imgui.Spacing()
	for i := 0; i < len(regs.Datastream); i++ {
		if i%2 != 0 {
			imgui.SameLine()
		}
		f := i
		imguiLabel(fmt.Sprintf("%2d.", f))
		label := fmt.Sprintf("##%dpointer", i)
		data := fmt.Sprintf("%08x", regs.Datastream[i].Pointer)
		if imguiHexInput(label, 8, &data) {
			win.img.dbg.PushFunction(func() {
				b := win.img.dbg.VCS().Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("datastream::%d::pointer", f), data)
			})
		}
	}
	imgui.EndGroup()

	imgui.SameLineV(0, 20)

	imgui.BeginGroup()
	imgui.Text("Increments")
	imgui.Spacing()
	for i := 0; i < len(regs.Datastream); i++ {
		if i%2 != 0 {
			imgui.SameLine()
		}
		f := i
		imguiLabel(fmt.Sprintf("%2d.", f))
		label := fmt.Sprintf("##m%dincrement", i)
		inc := fmt.Sprintf("%08x", regs.Datastream[i].Increment)
		if imguiHexInput(label, 8, &inc) {
			win.img.dbg.PushFunction(func() {
				b := win.img.dbg.VCS().Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("datastream::%d::increment", f), inc)
			})
		}
	}
	imgui.EndGroup()

	imguiSeparator()

	imguiLabel("Fast Fetch")
	ff := regs.FastFetch
	if imgui.Checkbox("##fastfetch", &ff) {
		win.img.dbg.PushFunction(func() {
			b := win.img.dbg.VCS().Mem.Cart.GetRegistersBus()
			b.PutRegister("fastfetch", fmt.Sprintf("%v", ff))
		})
	}

	imgui.SameLineV(0, 20)

	imguiLabel("Sample Mode")
	sm := regs.SampleMode
	if imgui.Checkbox("##samplemode", &sm) {
		win.img.dbg.PushFunction(func() {
			b := win.img.dbg.VCS().Mem.Cart.GetRegistersBus()
			b.PutRegister("samplemode", fmt.Sprintf("%v", sm))
		})
	}

	imguiSeparator()

	// loop over music fetchers
	imgui.Text("Music Fetchers")
	imgui.Spacing()
	for i := 0; i < len(regs.MusicFetcher); i++ {
		f := i

		imguiLabel(fmt.Sprintf("%d.", f))

		label := fmt.Sprintf("##m%dwaveform", i)
		waveform := fmt.Sprintf("%08x", regs.MusicFetcher[i].Waveform)
		imguiLabel("Waveform")
		if imguiHexInput(label, 8, &waveform) {
			win.img.dbg.PushFunction(func() {
				b := win.img.dbg.VCS().Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("music::%d::waveform", f), waveform)
			})
		}

		imgui.SameLine()
		label = fmt.Sprintf("##m%dfreq", i)
		freq := fmt.Sprintf("%08x", regs.MusicFetcher[i].Freq)
		imguiLabel("Freq")
		if imguiHexInput(label, 8, &freq) {
			win.img.dbg.PushFunction(func() {
				b := win.img.dbg.VCS().Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("music::%d::freq", f), freq)
			})
		}

		imgui.SameLine()
		label = fmt.Sprintf("##m%dcount", i)
		count := fmt.Sprintf("%08x", regs.MusicFetcher[i].Count)
		imguiLabel("Count")
		if imguiHexInput(label, 8, &count) {
			win.img.dbg.PushFunction(func() {
				b := win.img.dbg.VCS().Mem.Cart.GetRegistersBus()
				b.PutRegister(fmt.Sprintf("music::%d::count", f), count)
			})
		}
	}
}
