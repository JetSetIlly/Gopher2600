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

	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/imgui-go/v5"
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
	if selection := palette.draw(int(*pop.target)); selection != paletteNoSelection {
		*pop.target = uint8(selection)
		pop.callback()
		pop.state = popupClosed
	}

	imgui.End()
}

type palette struct {
	img *SdlImgui

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

const paletteNoSelection = -1

// returns the selected colour value or paletteNoSelection if no entry was
// clicked on
//
// use argument of paletteNoSelection to indicate that no palette entry
// should be shown to have been selected - selected entries will be drawn as a
// circle instead of a square
func (pal *palette) draw(selection int) int {
	var startDragAndDrop bool
	var showTooltip bool
	var hoverRGBA imgui.PackedColor

	selected := paletteNoSelection

	// step through all colours in palette
	for hue := 0; hue <= 0x0f; hue++ {
		p := imgui.CursorScreenPos()

		for lum := 0; lum <= 0x0e; lum += 2 {
			c := (hue << 4) | lum
			rgba := pal.img.getTVColour(uint8(c))
			hov, clck := pal.colRect(rgba, selection == c)
			if clck {
				selected = c
				startDragAndDrop = true
			}
			if hov {
				hoverRGBA = rgba
				showTooltip = true
			}
		}

		imgui.SetCursorScreenPos(
			p.Plus(imgui.Vec2{
				Y: pal.swatchSize - imgui.CurrentStyle().ItemSpacing().Y + pal.swatchGap,
			}),
		)
		imgui.Dummy(imgui.Vec2{})
	}

	if showTooltip {
		pal.img.imguiTooltip(func() {
			imgui.Text(fmt.Sprintf("#%06x", hoverRGBA&0x00ffffff))
		}, false)
	}

	if startDragAndDrop {
		imgui.PushStyleVarFloat(imgui.StyleVarPopupBorderSize, 0.0)
		imgui.PushStyleColor(imgui.StyleColorPopupBg, pal.img.cols.Transparent)
		if imgui.BeginDragDropSource(imgui.DragDropFlagsNone) {
			const paletteDragDropName = "PALETTE_DRAG_AND_DROP"
			imgui.SetDragDropPayload(paletteDragDropName, []byte{byte(selected)}, imgui.ConditionAlways)
			imgui.PushFont(pal.img.fonts.largeFontAwesome)
			imgui.Text(string(fonts.PaintRoller))
			imgui.PopFont()
			imgui.EndDragDropSource()
		}
		imgui.PopStyleColor()
		imgui.PopStyleVar()
	}

	return selected
}

func (pal *palette) colRect(col imgui.PackedColor, selected bool) (hover bool, clicked bool) {
	// position & dimensions of playfield bit
	p := imgui.CursorScreenPos()
	z := imgui.Vec2{X: pal.swatchSize, Y: pal.swatchSize}
	b := p.Plus(z)

	// if mouse is clicked in the range of the playfield bit
	mp := imgui.MousePos()
	hover = mp.X >= p.X && mp.X <= b.X && mp.Y >= p.Y && mp.Y <= b.Y
	clicked = hover && imgui.IsMouseClicked(0)

	dl := imgui.WindowDrawList()

	// show rectangle with color
	if selected {
		c := p.Plus(b).Times(0.5)
		dl.AddCircleFilled(c, pal.swatchSize*0.5, col)
	} else {
		dl.AddRectFilled(p, b, col)
	}

	// set up cursor for next widget
	z = imgui.Vec2{X: pal.swatchSize + pal.swatchGap}
	p = imgui.CursorScreenPos().Plus(z)
	imgui.SetCursorScreenPos(p)
	b = imgui.CursorScreenPos()
	imgui.Dummy(imgui.Vec2{})
	imgui.SetCursorScreenPos(b)

	return hover, clicked
}
