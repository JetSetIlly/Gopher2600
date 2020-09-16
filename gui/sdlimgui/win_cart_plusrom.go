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
	"github.com/inkyblackness/imgui-go/v2"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
)

const winCartPlusROMTitle = "PlusROM"

type winCartPlusROM struct {
	windowManagement
	widgetDimensions

	img *SdlImgui

	// ready flag colors
	colFlgReadyOn  imgui.PackedColor
	colFlgReadyOff imgui.PackedColor
}

func newWinCartPlusROM(img *SdlImgui) (managedWindow, error) {
	win := &winCartPlusROM{
		img: img,
	}

	return win, nil
}

func (win *winCartPlusROM) init() {
	win.widgetDimensions.init()
}

func (win *winCartPlusROM) destroy() {
}

func (win *winCartPlusROM) id() string {
	return winCartPlusROMTitle
}

func (win *winCartPlusROM) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.IsPlusROM {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{659, 35}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winCartPlusROMTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	host := win.img.lz.Cart.PlusROMAddrInfo.Host
	path := win.img.lz.Cart.PlusROMAddrInfo.Path

	imgui.AlignTextToFramePadding()
	imgui.Text("Hostname")
	imgui.SameLine()
	if imgui.InputText("##hostname", &host) {
		p := path
		win.img.lz.Dbg.PushRawEvent(func() {
			// because we're calling SetNetwork() in the debugger goroutine, we
			// have to get a fresh pointer to the PlusROM structure. we're
			// assuming that the type assertion will not fail
			//
			// also note that we've made another copy of the path string
			// because the first copy is to be used in the othe call to
			// InputText()
			win.img.lz.Dbg.VCS.Mem.Cart.GetContainer().(*plusrom.PlusROM).SetAddrInfo(host, p)
		})
	}

	imgui.AlignTextToFramePadding()
	imgui.Text("    Path")
	imgui.SameLine()
	if imgui.InputText("##path", &path) {
		win.img.lz.Dbg.PushRawEvent(func() {
			// see comment above. however note, that we *don't* need to make
			// another copy of host because the first copy has already been
			// used by this point so there is no chance of conflict (!)
			win.img.lz.Dbg.VCS.Mem.Cart.GetContainer().(*plusrom.PlusROM).SetAddrInfo(host, path)
		})
	}

	imgui.End()
}
