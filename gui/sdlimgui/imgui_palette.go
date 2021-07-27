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

type popupState int

const (
	popupClosed popupState = iota
	popupRequested
	popupActive
)

type popupPalette struct {
	img      *SdlImgui
	state    popupState
	target   *uint8
	callback func()

	// we set sizing information at time of request. it may be too early to do
	// this on popupPalette creation
	swatchSize float32
	swatchGap  float32

	// similarly, the palette to use will be decided at request time
	palette     packedPalette
	paletteName string

	pos imgui.Vec2
	cnt imgui.Vec2
}

func newPopupPalette(img *SdlImgui) *popupPalette {
	pal := &popupPalette{
		img: img,
	}
	return pal
}

func (pal *popupPalette) request(target *uint8, callback func()) {
	pal.state = popupRequested
	pal.target = target
	pal.callback = callback
	pal.swatchSize = imgui.FrameHeight() * 0.75
	pal.swatchGap = pal.swatchSize * 0.1
	pal.pos = imgui.MousePos()
	pal.paletteName, pal.palette, _ = pal.img.imguiTVPalette()
	pal.cnt = pal.img.imguiWindowQuadrant(pal.pos)
}

func (pal *popupPalette) draw() {
	if pal.state == popupClosed {
		return
	}

	// we need to filter out the remnants of the click that caused this popup
	// window to open. if we're still in the popupRequested state then move to
	// the popupActive state when the mouse button has been released
	if pal.state == popupRequested {
		if imgui.IsMouseReleased(0) {
			pal.state = popupActive
		}
		return
	}

	imgui.SetNextWindowPosV(pal.pos, 0, pal.cnt)
	imgui.BeginV("Palette", nil, imgui.WindowFlagsNoDecoration|imgui.WindowFlagsNoMove|imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoSavedSettings)

	// close window if mouse clicked outside of window
	if !imgui.IsWindowHovered() && imgui.IsMouseClicked(0) {
		pal.state = popupClosed
	}

	// information bar
	imgui.Text("Current: ")
	imgui.SameLine()
	imgui.Text(pal.paletteName)
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("%02x", *pal.target))
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("#%06x", pal.palette[*pal.target]&0x00ffffff))

	imgui.Spacing()

	// step through all colours in palette
	for i := 0; i < len(pal.palette); i++ {
		if pal.colRect(uint8(i)) {
			*pal.target = uint8(i)
			if pal.callback != nil {
				pal.callback()
			}
			pal.state = popupClosed
		}

		// start a new row every 16 swatches
		if (i+1)%16 == 0 {
			p := imgui.CursorScreenPos()
			p.Y += pal.swatchSize + pal.swatchGap
			p.X -= 16 * (pal.swatchSize + pal.swatchGap)
			imgui.SetCursorScreenPos(p)
		}
	}

	imgui.End()
}

func (pal *popupPalette) colRect(col uint8) bool {
	c := pal.palette[col]

	// position & dimensions of playfield bit
	a := imgui.CursorScreenPos()
	b := a
	b.X += pal.swatchSize
	b.Y += pal.swatchSize

	mp := imgui.MousePos()
	hover := mp.X >= a.X && mp.X <= b.X && mp.Y >= a.Y && mp.Y <= b.Y

	// if mouse is clicked in the range of the playfield bit
	clicked := hover && imgui.IsMouseClicked(0)

	// tooltip
	if hover {
		imguiTooltip(func() {
			imgui.Text(fmt.Sprintf("%02x", col))
			imgui.SameLine()
			imgui.Text(fmt.Sprintf("#%06x", c&0x00ffffff))
		}, false)
	}

	// show rectangle with color
	dl := imgui.WindowDrawList()
	dl.AddRectFilled(a, b, c)

	// set up cursor for next widget
	a.X += pal.swatchSize + pal.swatchGap
	imgui.SetCursorScreenPos(a)

	return clicked
}
