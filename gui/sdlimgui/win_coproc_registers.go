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
)

const winCoProcRegistersID = "Registers"
const winCoProcRegistersMenu = "Registers"

type winCoProcRegisters struct {
	debuggerWin

	img *SdlImgui

	showSrcInTooltip bool
	optionsHeight    float32
}

func newWinCoProcRegisters(img *SdlImgui) (window, error) {
	win := &winCoProcRegisters{
		img:              img,
		showSrcInTooltip: true,
	}
	return win, nil
}

func (win *winCoProcRegisters) init() {
}

func (win *winCoProcRegisters) id() string {
	return winCoProcRegistersID
}

func (win *winCoProcRegisters) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	// do not open window if there is no coprocessor available
	coproc := win.img.cache.VCS.Mem.Cart.GetCoProc()
	if coproc == nil {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{942, 97}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{520, 390}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{400, 300}, imgui.Vec2{551, 1000})

	title := fmt.Sprintf("%s %s", coproc.ProcessorID(), winCoProcRegistersID)
	if imgui.BeginV(win.debuggerID(title), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCoProcRegisters) draw() {
	// coproc := win.img.cq.vcs.Mem.Cart.GetCoProc()
	// spec := coproc.RegisterSpec()

	// for _, regs := range spec {
	// 	drawRegGroup := func() {
	// 		for r := regs.Start; r <= regs.End; r++ {
	// 			if v, f, ok := coproc.RegisterFormatted(r); ok {
	// 				_ = v
	// 				imgui.InputText(regs.Label(r), &f)
	// 			}
	// 		}
	// 	}

	// 	if regs.Name == coprocessor.ExtendedRegisterCoreGroup {
	// 		drawRegGroup()
	// 	} else {
	// 		if imgui.CollapsingHeader(regs.Name) {
	// 			drawRegGroup()
	// 		}
	// 	}

	// }

	// _ = spec
}
