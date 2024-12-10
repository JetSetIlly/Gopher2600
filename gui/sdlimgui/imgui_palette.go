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
	name     string

	pos imgui.Vec2
	cnt imgui.Vec2
}

func newPopupPalette(img *SdlImgui) *popupPalette {
	pop := &popupPalette{
		img: img,
	}
	return pop
}

func (pop *popupPalette) request(target *uint8, callback func()) {
	pop.state = popupRequested
	pop.target = target
	pop.callback = callback
	pop.name = pop.img.cache.TV.GetFrameInfo().Spec.ID
	pop.pos = imgui.MousePos()
	pop.cnt = pop.img.imguiWindowQuadrant(pop.pos)
}

func (pop *popupPalette) draw() {
	if pop.state == popupClosed {
		return
	}

	// we need to filter out the remnants of the click that caused this popup
	// window to open. if we're still in the popupRequested state then move to
	// the popupActive state when the mouse button has been released
	if pop.state == popupRequested {
		if imgui.IsMouseReleased(0) {
			pop.state = popupActive
		}
		return
	}

	imgui.SetNextWindowPosV(pop.pos, 0, pop.cnt)
	imgui.BeginV("Palette", nil, imgui.WindowFlagsNoDecoration|imgui.WindowFlagsNoMove|imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoSavedSettings)

	// close window if mouse clicked outside of window
	if !imgui.IsWindowHovered() && imgui.IsMouseClicked(0) {
		pop.state = popupClosed
	}

	// information bar
	imgui.Text("Current: ")
	imgui.SameLine()
	imgui.Text(pop.name)
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("%02x", *pop.target))

	imgui.Spacing()

	palette := newPalette(pop.img)
	if selection, ok := palette.draw(int(*pop.target)); ok {
		*pop.target = uint8(selection)
		pop.callback()
		pop.state = popupClosed
	}

	imgui.End()
}

type palette struct {
	img *SdlImgui

	// we set sizing information at time of request. it may be too early to do
	// this on popupPalette creation
	swatchSize float32
	swatchGap  float32
}

func newPalette(img *SdlImgui) *palette {
	pal := &palette{
		img: img,
	}

	pal.swatchSize = imgui.FrameHeight() * 0.75
	pal.swatchGap = pal.swatchSize * 0.1

	return pal
}

const paletteDragDropName = "PALETTE"

func (pal *palette) draw(selection int) (int, bool) {
	val := -1

	// step through all colours in palette
	for hue := 0; hue <= 0x0f; hue++ {
		for lum := 0; lum <= 0x0e; lum += 2 {
			c := (hue << 4) | lum

			if pal.colRect(c, pal.img.getTVColour(uint8(c)), selection == c) {
				val = c

				imgui.PushStyleVarFloat(imgui.StyleVarPopupBorderSize, 0.0)
				imgui.PushStyleColor(imgui.StyleColorPopupBg, pal.img.cols.Transparent)
				if imgui.BeginDragDropSource(imgui.DragDropFlagsNone) {
					imgui.SetDragDropPayload(paletteDragDropName, []byte{byte(c)}, imgui.ConditionAlways)
					imgui.PushFont(pal.img.fonts.largeFontAwesome)
					imgui.Text(string(fonts.PaintRoller))
					imgui.PopFont()
					imgui.EndDragDropSource()
				}
				imgui.PopStyleColor()
				imgui.PopStyleVar()
			}
		}

		p := imgui.CursorScreenPos()
		p.Y += pal.swatchSize + pal.swatchGap
		p.X -= 8 * (pal.swatchSize + pal.swatchGap)
		imgui.SetCursorScreenPos(p)
	}

	return val, val != -1
}

func (pal *palette) colRect(idx int, col imgui.PackedColor, selected bool) bool {
	pal.img.rnd.pushTVColor()
	defer pal.img.rnd.popTVColor()

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
		pal.img.imguiTooltip(func() {
			imgui.Text(fmt.Sprintf("%02x", idx))
			imgui.SameLine()
			imgui.Text(fmt.Sprintf("#%06x", col&0x00ffffff))
		}, false)
	}

	dl := imgui.WindowDrawList()

	// show rectangle with color
	if selected {
		c := a.Plus(b).Times(0.5)
		dl.AddCircleFilled(c, pal.swatchSize*0.5, col)
	} else {
		dl.AddRectFilled(a, b, col)
	}

	// set up cursor for next widget
	a.X += pal.swatchSize + pal.swatchGap
	imgui.SetCursorScreenPos(a)

	return clicked
}
