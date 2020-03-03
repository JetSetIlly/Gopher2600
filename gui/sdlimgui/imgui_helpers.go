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

import (
	"github.com/inkyblackness/imgui-go/v2"
)

// requires the minimum Vec2{} required to fit any of the string values
// listed in the arguments
func imguiGetFrameDim(s string, t ...string) imgui.Vec2 {
	w := imgui.CalcTextSize(s, false, 0)
	for i := range t {
		y := imgui.CalcTextSize(t[i], false, 0)
		if y.X > w.X {
			w = y
		}
	}
	w.Y = imgui.FontSize() + (imgui.CurrentStyle().FramePadding().Y * 2.0)
	return w
}

// draw toggle button at current cursor position
func imguiToggleButton(id string, v *bool, col imgui.Vec4) {
	bg := imgui.PackedColorFromVec4(col)
	p := imgui.CursorScreenPos()
	dl := imgui.WindowDrawList()

	height := imgui.FrameHeight()
	width := height * 1.55
	radius := height * 0.50
	t := float32(0.0)
	if *v {
		t = 1.0
	}

	// const animSpeed = 0.08
	// ctx, _ := imgui.CurrentContext()
	// if ctx.LastActiveId == ctx.CurrentWindow.GetID(id) {
	// 	tanim := ctx.LastActiveIdTimer / animSpeed
	// 	if tanim < 0.0 {
	// 		tanim = 0.0
	// 	} else if tanim > 1.0 {
	// 		tanim = 1.0
	// 	}
	// 	if *v {
	// 		t = tanim
	// 	} else {
	// 		t = 1.0 - tanim
	// 	}
	// }

	imgui.InvisibleButtonV(id, imgui.Vec2{width, height})
	if imgui.IsItemClicked() {
		*v = !*v
	}

	dl.AddRectFilledV(p, imgui.Vec2{p.X + width, p.Y + height}, bg, radius, imgui.DrawCornerFlagsAll)
	dl.AddCircleFilled(imgui.Vec2{p.X + radius + t*(width-radius*2.0), p.Y + radius},
		radius-1.5, imgui.PackedColorFromVec4(imgui.Vec4{1.0, 1.0, 1.0, 1.0}))
}

// input text that accepts a maximum number of hex digits
func imguiHexInput(label string, aggressiveUpdate bool, digits int, content *string, update func()) {
	cb := func(d imgui.InputTextCallbackData) int32 {
		b := string(d.Buffer())

		// restrict length of input to two characters. note that restriction to
		// hexadecimal characters is handled by imgui's CharsHexadecimal flag
		// given to InputTextV()
		if len(b) > digits {
			d.DeleteBytes(0, len(b))
			b = b[:digits]
			d.InsertBytes(0, []byte(b))
			d.MarkBufferModified()
		}

		return 0
	}

	// flags used with InputTextV()
	flags := imgui.InputTextFlagsCharsHexadecimal |
		imgui.InputTextFlagsCallbackAlways |
		imgui.InputTextFlagsAutoSelectAll

	// with aggressiveUpdate the values entered will be given to the onEnter()
	// function immediately and not just when the enter key is pressed.
	if aggressiveUpdate {
		flags |= imgui.InputTextFlagsEnterReturnsTrue
	}

	if imgui.InputTextV(label, content, flags, cb) {
		update()
	}
}

// calls Text but preceeds it with AlignTextToFramePadding() and follows it
// with SameLine(). a common enought pattern to warrent a function call
func imguiLabel(label string) {
	imgui.AlignTextToFramePadding()
	imgui.Text(label)
	imgui.SameLine()
}

// returns a Vec2 suitable for use as a position vector when opening a imgui
// window. The X and Y are set such that 0.0 <= value <= 1.0
//
// the vector is weighted to reflect the quadrant in which the supplied
// argument p (mouse position, say) falls within.
func (img *SdlImgui) imguiWindowQuadrant(p imgui.Vec2) imgui.Vec2 {
	sp := img.wm.screenPos
	ww, wh := img.plt.window.GetSize()

	q := imgui.Vec2{X: 0.25, Y: 0.25}

	if p.X > (sp.X+float32(ww))/2 {
		q.X = 0.75
	}

	if p.Y > (sp.Y+float32(wh))/2 {
		q.Y = 0.75
	}

	return q
}

// use appropriate palette for television spec
func (img *SdlImgui) imguiPackedPalette() (string, packedPalette) {
	switch img.tv.GetSpec().ID {
	case "PAL":
		return img.tv.GetSpec().ID, img.cols.packedPalettePAL
	case "NTSC":
		return img.tv.GetSpec().ID, img.cols.packedPaletteNTSC
	}

	return "NTSC?", img.cols.packedPaletteNTSC
}

// draw swatch. returns true if clicked. a good response to a click event is to
// open up an instance of popupPalette
func (img *SdlImgui) imguiSwatch(col uint8) (clicked bool) {
	_, pal := img.imguiPackedPalette()
	c := pal[col]

	// position & dimensions of swatch
	r := imgui.FontSize() * 0.75
	p := imgui.CursorScreenPos()
	p.X += r
	p.Y += r

	// if mouse is clicked in the range of the swatch. very simple detection,
	// not accounting for the fact that the swatch is visibly circular
	if imgui.IsMouseClicked(0) {
		pos := imgui.MousePos()
		clicked = pos.X >= p.X-r && pos.X <= p.X+r && pos.Y >= p.Y-r && pos.Y <= p.Y+r
	}

	// draw swatch
	dl := imgui.WindowDrawList()
	dl.AddCircleFilled(p, r, c)

	// set up cursor for next widget
	p.X += 2 * r
	p.Y -= r
	imgui.SetCursorScreenPos(p)

	return clicked
}

// drawlistSequence provides a neat way of drawlist elements of a uniform size in
// sequence
type drawlistSequence struct {
	img     *SdlImgui
	palette packedPalette
	size    imgui.Vec2
	spacing imgui.Vec2

	startX float32
	prevX  float32
	prevY  float32

	nextItemSameLine bool
}

// create and start a new sequence. spacing is expressed as fraction of the
// current FontSize()
func newDrawlistSequence(img *SdlImgui, size imgui.Vec2, spacing float32) *drawlistSequence {
	seq := &drawlistSequence{
		img:     img,
		size:    size,
		spacing: imgui.Vec2{X: imgui.FontSize() * spacing, Y: imgui.FontSize() * spacing},
	}
	_, seq.palette = img.imguiPackedPalette()
	seq.start()
	return seq
}

// start resets the reference position. convenient to use if size/spacing is not changing
func (seq *drawlistSequence) start() {
	seq.prevX = imgui.WindowPos().X + imgui.CursorPosX()
	seq.prevY = imgui.WindowPos().Y + imgui.CursorPosY() - seq.size.Y - seq.spacing.Y
	seq.startX = seq.prevX
	seq.nextItemSameLine = false
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

func (seq *drawlistSequence) rectFilled(col uint8) (clicked bool) {
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

	// get color
	c := seq.palette[col]

	// position & dimensions of playfield bit
	a := imgui.Vec2{X: x, Y: y}
	b := a
	b.X += seq.size.X
	b.Y += seq.size.Y

	// if mouse is clicked in the range of the playfield bit
	if imgui.IsMouseClicked(0) {
		pos := imgui.MousePos()
		clicked = pos.X >= a.X && pos.X <= b.X && pos.Y >= a.Y && pos.Y <= b.Y
	}

	// draw square
	dl := imgui.WindowDrawList()
	dl.AddRectFilled(a, b, c)

	// record coordinates for use by next element
	seq.prevX = a.X
	seq.prevY = a.Y

	// set cursor position for any non colorSequence widgets
	imgui.SetCursorScreenPos(imgui.Vec2{X: x + seq.size.X + seq.spacing.X, Y: y})

	return clicked
}
