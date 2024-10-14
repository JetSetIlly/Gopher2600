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
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui/fonts"
)

const winPaintID = "Paint"

type winPaint struct {
	debuggerWin
	img           *SdlImgui
	palette       *palette
	selectedColor int
}

func newWinPaint(img *SdlImgui) (window, error) {
	win := &winPaint{
		img: img,
	}

	return win, nil
}

func (win *winPaint) init() {
	win.palette = newPalette(win.img)
}

func (win *winPaint) id() string {
	return winPaintID
}

func (win *winPaint) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 756, Y: 117}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	imgui.End()

	return true
}

const painDragDrop = "PAINT"

func (win *winPaint) draw() {
	if imgui.BeginTable("drawToolbox", 2) {
		imgui.TableNextRow()
		imgui.TableNextColumn()
		imgui.PushFont(win.img.fonts.largeFontAwesome)
		imgui.PushStyleVarFloat(imgui.StyleVarFrameBorderSize, 1.0)
		imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{X: 5.0, Y: 10.0})

		imgui.Button(string(fonts.PaintRoller))
		if imgui.BeginDragDropSource(imgui.DragDropFlagsNone) {
			imgui.SetDragDropPayload(painDragDrop, []byte{byte(win.selectedColor)}, imgui.ConditionAlways)
			imgui.Text(string(fonts.PaintRoller))
			imgui.EndDragDropSource()
		}

		imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
		imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
		imgui.Button(string(fonts.PaintBrush))
		imgui.Button(string(fonts.Pencil))
		imgui.PopStyleVar()
		imgui.PopItemFlag()

		imgui.PopStyleVarV(2)
		imgui.PopFont()

		imgui.TableNextColumn()
		if selection, ok := win.palette.draw(win.selectedColor); ok {
			win.selectedColor = selection
		}

		imgui.EndTable()
	}
}
