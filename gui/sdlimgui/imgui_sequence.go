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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlimgui

import "github.com/inkyblackness/imgui-go/v2"

// drawlistSequence provides a neat way of drawlist elements of a uniform size in
// sequence
type drawlistSequence struct {
	img              *SdlImgui
	palette          packedPalette
	size             imgui.Vec2
	spacing          imgui.Vec2
	depressionAmount float32

	startX float32
	prevX  float32
	prevY  float32

	nextItemSameLine  bool
	nextItemDepressed bool
}

// create and start a new sequence. spacing is expressed as fraction of the
// current FontSize()
func newDrawlistSequence(img *SdlImgui, size imgui.Vec2, spacing float32) *drawlistSequence {
	seq := &drawlistSequence{
		img:              img,
		size:             size,
		spacing:          imgui.Vec2{X: imgui.FontSize() * spacing, Y: imgui.FontSize() * spacing},
		depressionAmount: 2.0,
	}
	_, seq.palette = img.imguiTVPalette()
	seq.start()
	return seq
}

// start resets the reference position. convenient to use if size/spacing is not changing.
// returns starting X position for future reference, if required
//
// should be coupled with a call to end()
func (seq *drawlistSequence) start() float32 {
	seq.prevX = imgui.CursorScreenPos().X
	seq.prevY = imgui.CursorScreenPos().Y - seq.size.Y - seq.spacing.Y
	seq.startX = seq.prevX
	seq.nextItemSameLine = false
	seq.nextItemDepressed = false
	imgui.BeginGroup()
	return seq.startX
}

// every call to start() should be coupled with a call to end()
func (seq *drawlistSequence) end() {
	imgui.EndGroup()

}

// calling sameLine() before a call to element may not have the effect you
// expect. avoid calling the function until at least one element has been
// drawn.
func (seq *drawlistSequence) sameLine() {
	seq.nextItemSameLine = true
}

// returns the X value that is in the middle of the n'th element
func (seq *drawlistSequence) offsetX(n int) float32 {
	return seq.startX + float32(n)*(seq.size.X+seq.spacing.X) + seq.size.X*0.5
}

func (seq *drawlistSequence) rectFillTvCol(col uint8) (clicked bool) {
	return seq.rectFill(seq.palette[col])
}

func (seq *drawlistSequence) rectFill(col imgui.PackedColor) (clicked bool) {
	var x, y float32

	if seq.nextItemSameLine {
		x = seq.prevX + seq.size.X + seq.spacing.X
		y = seq.prevY
	} else {
		x = seq.startX
		y = seq.prevY + seq.size.Y + seq.spacing.Y
	}

	// reset sameline flag
	seq.nextItemSameLine = false

	// position & dimensions of playfield bit
	a := imgui.Vec2{X: x, Y: y}
	b := a
	b.X += seq.size.X
	b.Y += seq.size.Y

	// if mouse is clicked in the range of the playfield bit
	if imgui.IsWindowHovered() && imgui.IsMouseClicked(0) {
		pos := imgui.MousePos()
		clicked = pos.X >= a.X && pos.X <= b.X && pos.Y >= a.Y && pos.Y <= b.Y
	}

	// draw square
	dl := imgui.WindowDrawList()

	if seq.nextItemDepressed {
		seq.nextItemDepressed = false
		a.X += seq.depressionAmount
		a.Y += seq.depressionAmount
		b.X -= seq.depressionAmount
		b.Y -= seq.depressionAmount
		dl.AddRectFilled(a, b, col)
		a.X -= seq.depressionAmount
		a.Y -= seq.depressionAmount
	} else {
		dl.AddRectFilled(a, b, col)
	}

	// record coordinates for use by next element
	seq.prevX = a.X
	seq.prevY = a.Y

	// set cursor position for any non colorSequence widgets
	imgui.SetCursorScreenPos(imgui.Vec2{X: x + seq.size.X + seq.spacing.X, Y: y})

	return clicked
}

func (seq *drawlistSequence) rectEmpty(col imgui.PackedColor) {
	var x, y float32

	if seq.nextItemSameLine {
		x = seq.prevX + seq.size.X + seq.spacing.X
		y = seq.prevY
	} else {
		x = seq.startX
		y = seq.prevY + seq.size.Y + seq.spacing.Y
	}

	// reset sameline flag
	seq.nextItemSameLine = false

	// position & dimensions of playfield bit
	a := imgui.Vec2{X: x, Y: y}
	b := a
	b.X += seq.size.X
	b.Y += seq.size.Y

	// draw square
	dl := imgui.WindowDrawList()

	if seq.nextItemDepressed {
		seq.nextItemDepressed = false
		a.X += seq.depressionAmount
		a.Y += seq.depressionAmount
		b.X -= seq.depressionAmount
		b.Y -= seq.depressionAmount
		dl.AddRect(a, b, col)
		a.X -= seq.depressionAmount
		a.Y -= seq.depressionAmount
	} else {
		dl.AddRect(a, b, col)
	}

	// record coordinates for use by next element
	seq.prevX = a.X
	seq.prevY = a.Y

	// set cursor position for any non colorSequence widgets
	imgui.SetCursorScreenPos(imgui.Vec2{X: x + seq.size.X + seq.spacing.X, Y: y})
}
