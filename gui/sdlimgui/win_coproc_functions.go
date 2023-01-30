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
	"github.com/jetsetilly/gopher2600/coprocessor/developer"
)

const winCoProcFunctionsID = "Coprocessor Functions"
const winCoProcFunctionsMenu = "Functions"

type winCoProcFunctions struct {
	debuggerWin

	img *SdlImgui

	showSrcInTooltip bool
	optionsHeight    float32
}

func newWinCoProcFunctions(img *SdlImgui) (window, error) {
	win := &winCoProcFunctions{
		img: img,
	}
	return win, nil
}

func (win *winCoProcFunctions) init() {
}

func (win *winCoProcFunctions) id() string {
	return winCoProcFunctionsID
}

func (win *winCoProcFunctions) debuggerDraw() {
	if !win.debuggerOpen {
		return
	}

	if !win.img.lz.Cart.HasCoProcBus {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{775, 102}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{400, 655}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{300, 400}, imgui.Vec2{600, 1000})

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.CoProcID, winCoProcFunctionsID)
	if imgui.BeginV(win.debuggerID(title), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()
}

func (win *winCoProcFunctions) draw() {
	win.img.dbg.CoProcDev.BorrowSource(func(src *developer.Source) {
		if src == nil {
			imgui.Text("No source files available")
			return
		}

		if len(src.FunctionNames) == 0 {
			imgui.Text("No functions defined")
			return
		}

		for _, n := range src.FunctionNames {
			fn := src.Functions[n]
			if !fn.IsStub() {
				if imgui.Selectable(n) {
					srcWin := win.img.wm.debuggerWindows[winCoProcSourceID].(*winCoProcSource)
					srcWin.gotoSourceLine(fn.DeclLine)
				}
			}
		}
	})
}
