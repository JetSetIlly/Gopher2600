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
	"image"

	"github.com/inkyblackness/imgui-go/v4"
)

const magnifyMax = 3
const magnifyMin = 10

type dbgScrMagnify struct {
	// textures
	tooltipTexture uint32
	windowTexture  uint32

	// whether to show magnification in the tooltip
	showInTooltip bool

	// area of magnification for tooltip
	tooltipClip image.Rectangle

	// the amount of zoom in the tooltip magnification
	tooltipZoom int

	// whether magnification window is open
	windowOpen bool

	// area of magnification for window
	windowClip image.Rectangle

	// centre point of magnification area for window
	windowClipCenter dbgScrMousePoint

	// the amount of zoom in the magnify window
	windowZoom int
}

func (mag *dbgScrMagnify) setWindowClip(mouse dbgScrMouse) {
	mag.windowOpen = true
	mag.windowClipCenter = mouse.scaled
	mag.windowClip = image.Rect(mag.windowClipCenter.x-mag.windowZoom,
		mag.windowClipCenter.y-mag.windowZoom*pixelWidth,
		mag.windowClipCenter.x+mag.windowZoom,
		mag.windowClipCenter.y+mag.windowZoom*pixelWidth)
}

func (mag *dbgScrMagnify) zoomWindowClip(delta float32) {
	if delta < 0 && mag.windowZoom < magnifyMin {
		mag.windowZoom++
	} else if delta > 0 && mag.windowZoom > magnifyMax {
		mag.windowZoom--
	}
	mag.windowClip = image.Rect(mag.windowClipCenter.x-mag.windowZoom,
		mag.windowClipCenter.y-mag.windowZoom*pixelWidth,
		mag.windowClipCenter.x+mag.windowZoom,
		mag.windowClipCenter.y+mag.windowZoom*pixelWidth)
}

func (mag *dbgScrMagnify) drawWindow(io *imgui.IO, cols *imguiColors) {
	if !mag.windowOpen {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{8, 28}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{200, 200}, imgui.ConditionFirstUseEver)

	if imgui.BeginV("Magnification", &mag.windowOpen, imgui.WindowFlagsNoScrollbar) {
		sz := imgui.ContentRegionAvail()
		if sz.X >= sz.Y {
			imgui.SetCursorPos(imgui.CursorPos().Plus(imgui.Vec2{(sz.X - sz.Y) / 2.0, 0}))
			sz = imgui.Vec2{sz.Y, sz.Y}
		} else {
			imgui.SetCursorPos(imgui.CursorPos().Plus(imgui.Vec2{0, (sz.Y - sz.X) / 2.0}))
			sz = imgui.Vec2{sz.X, sz.X}
		}

		imgui.PushStyleColor(imgui.StyleColorButton, cols.Transparent)
		imgui.PushStyleColor(imgui.StyleColorButtonActive, cols.Transparent)
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, cols.Transparent)
		imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{0.0, 0.0})
		imgui.ImageButton(imgui.TextureID(mag.windowTexture), sz)

		if imgui.IsItemHovered() {
			_, delta := io.MouseWheel()
			if delta != 0 {
				mag.zoomWindowClip(delta)
			}
		}

		imgui.PopStyleVar()
		imgui.PopStyleColorV(3)
	}

	imgui.End()
}

func (mag *dbgScrMagnify) drawTooltip(io *imgui.IO, mouse dbgScrMouse) {
	if !mag.showInTooltip {
		return
	}
	_, delta := io.MouseWheel()
	if delta < 0 && mag.tooltipZoom < magnifyMin {
		mag.tooltipZoom++
	} else if delta > 0 && mag.tooltipZoom > magnifyMax {
		mag.tooltipZoom--
	}

	mag.tooltipClip = image.Rect(mouse.scaled.x-mag.tooltipZoom,
		mouse.scaled.y-mag.tooltipZoom*pixelWidth,
		mouse.scaled.x+mag.tooltipZoom,
		mouse.scaled.y+mag.tooltipZoom*pixelWidth)

	imgui.Image(imgui.TextureID(mag.tooltipTexture), imgui.Vec2{200, 200})
	imguiSeparator()
}
