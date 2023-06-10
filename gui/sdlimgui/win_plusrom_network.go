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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
)

const winPlusROMNetworkID = "PlusROM Network"
const winPlusROMNetworkMenu = "Network"

type winPlusROMNetwork struct {
	debuggerWin

	img *SdlImgui
}

func newWinPlusROMNetwork(img *SdlImgui) (window, error) {
	win := &winPlusROMNetwork{
		img: img,
	}

	return win, nil
}

func (win *winPlusROMNetwork) init() {
}

func (win *winPlusROMNetwork) id() string {
	return winPlusROMNetworkID
}

func (win *winPlusROMNetwork) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	if !win.img.lz.Cart.IsPlusROM {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{659, 35}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winPlusROMNetwork) draw() {
	host := win.img.lz.Cart.PlusROMAddrInfo.Host
	path := win.img.lz.Cart.PlusROMAddrInfo.Path
	send := win.img.lz.Cart.PlusROMSendState

	imgui.AlignTextToFramePadding()
	imgui.Text("Hostname")
	imgui.SameLine()
	imgui.PushItemWidth(imguiRemainingWinWidth())
	if imgui.InputText("##hostname", &host) {
		win.img.term.pushCommand(fmt.Sprintf("PLUSROM HOST %s", host))
	}
	imgui.PopItemWidth()

	imgui.AlignTextToFramePadding()
	imgui.Text("    Path")
	imgui.SameLine()
	imgui.PushItemWidth(imguiRemainingWinWidth())
	if imgui.InputText("##path", &path) {
		win.img.term.pushCommand(fmt.Sprintf("PLUSROM PATH %s", path))
	}
	imgui.PopItemWidth()

	if send.Cycles > 0 {
		imgui.PushFont(win.img.glsl.fonts.gopher2600Icons)
		imgui.Text(fmt.Sprintf("%c", fonts.Wifi))
		imgui.PopFont()
	}

	imguiSeparator()

	if imgui.CollapsingHeaderV("Send Buffer", imgui.TreeNodeFlagsDefaultOpen) {
		alpha := false

		before := func(idx int) {
		}

		after := func(idx int) {
			if send.Cycles > 0 {
				if idx == int(send.SendLen) {
					imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
					alpha = true
				}
			} else if idx == int(send.WritePtr) {
				imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
				alpha = true
			}
		}

		commit := func(idx int, value uint8) {
			win.img.dbg.PushFunction(func() {
				win.img.vcs.Mem.Cart.GetContainer().(*plusrom.PlusROM).SetSendBuffer(idx, value)
			})
		}

		drawByteGrid("pluscartsendbuffer", send.Buffer[:], 0, before, after, commit)
		if alpha {
			imgui.PopStyleVar()
		}
	}
}
