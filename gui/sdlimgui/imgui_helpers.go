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
	"image/color"
	"strconv"
	"strings"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// return the height of the window from the current cursor position to the end
// of the window frame. useful for calculating scroll areas for windows with a
// static header.
//
// the height of a static footer must be subtracted from the returned value.
// the measuredHeight() function is useful for measuring footers.
func imguiRemainingWinHeight() float32 {
	return imgui.WindowHeight() - imgui.CursorPosY() - imgui.CurrentStyle().FramePadding().Y*2 - imgui.CurrentStyle().ItemInnerSpacing().Y
}

// return the width of the window from the current cursor position to the edge
// of the window frame. subtracts padding and spacing from the edge to make it
// suitable for sizing buttons etc.
func imguiRemainingWinWidth() float32 {
	w := imgui.WindowWidth() - imgui.CursorPosX()
	w -= imgui.CurrentStyle().FramePadding().X + imgui.CurrentStyle().ItemInnerSpacing().X
	return w
}

// divide remaining win width by n taking into account spacing between
// widgets. useful for cheap tabulation of buttons.
func imguiDivideWinWidth(n int) float32 {
	w := imguiRemainingWinWidth() / float32(n)
	w -= imgui.CurrentStyle().FramePadding().X
	return w
}

// returns the minimum Vec2{} required to fit any of the string values listed
// in the arguments.
func imguiGetFrameDim(s string, t ...string) imgui.Vec2 {
	w := imgui.CalcTextSize(s, false, 0)
	for i := range t {
		y := imgui.CalcTextSize(t[i], false, 0)
		if y.X > w.X {
			w = y
		}
	}

	w.Y = imgui.FontSize() + (imgui.CurrentStyle().FramePadding().Y * 2.5)

	// comboboxes in particuar look better with a small amount of trailing space
	w.X += imgui.CurrentStyle().FramePadding().X * 2.5

	return w
}

// returns the pixel width of a text string length characters wide. assumes all
// characters are of the same width. Uses the 'X' character for measurement.
func imguiTextWidth(length int) float32 {
	if length < 1 {
		return 0
	}
	return imguiGetFrameDim(strings.Repeat("X", length)).X
}

// return coordinates for right alignment of a string to previous imgui widget.
// func imguiRightAlign(s string) imgui.Vec2 {
// 	// this dearimgui dance gets the X position of the end of the last widget.
// 	// leaving us with c, a Vec2 with the correct Y position
// 	c := imgui.CursorPos()
// 	imgui.SameLine()
// 	x := imgui.CursorPosX()
// 	imgui.SetCursorPos(c)
// 	c = imgui.CursorPos()

// 	// the X coordinate can be set by subtracting the width of the text from
// 	// the stored x value
// 	c.X = x - imguiTextWidth(len(s)) + imgui.CurrentStyle().FramePadding().X

// 	return c
// }

// adds min/max indicators to imgui.SliderInt. returns true if slider has changed.
// func imguiSliderInt(label string, f *int32, s int32, e int32) bool {
// 	v := imgui.SliderInt(label, f, s, e)

// 	// alignment information for frame number indicators below
// 	min := fmt.Sprintf("%d", s)
// 	max := fmt.Sprintf("%d", e)
// 	align := imguiRightAlign(max)

// 	// rewind frame information
// 	imgui.Text(min)
// 	imgui.SameLine()
// 	imgui.SetCursorPos(align)
// 	imgui.Text(max)

// 	return v
// }

// draw toggle button at current cursor position. returns true if toggle has been clicked.
func imguiToggleButton(id string, v bool, col imgui.Vec4) bool {
	height := imgui.FrameHeight() * 0.75
	width := height * 1.55
	radius := height * 0.50
	t := float32(0.0)
	if v {
		t = 1.0
	}

	bg := imgui.PackedColorFromVec4(col)
	p := imgui.CursorScreenPos().Plus(imgui.Vec2{X: 0, Y: (imgui.FrameHeight() / 2) - (height / 2)})
	dl := imgui.WindowDrawList()

	r := false

	imgui.InvisibleButtonV(id, imgui.Vec2{width, height}, imgui.ButtonFlagsMouseButtonLeft)
	if imgui.IsItemClicked() {
		r = true
	}

	dl.AddRectFilledV(p, imgui.Vec2{p.X + width, p.Y + height}, bg, radius, imgui.DrawCornerFlagsAll)
	dl.AddCircleFilled(imgui.Vec2{p.X + radius + t*(width-radius*2.0), p.Y + radius},
		radius-1.5, imgui.PackedColorFromVec4(imgui.Vec4{1.0, 1.0, 1.0, 1.0}))

	return r
}

// draw vertical toggle button at current cursor position. returns true if toggle has been clicked.
//
// NOTE: this function has been hacked to work with the status register toggles
// in win_cpu. any changes to this function will have to bear that in mind.
func imguiToggleButtonVertical(id string, v bool, col imgui.Vec4) bool {
	bg := imgui.PackedColorFromVec4(col)
	p := imgui.CursorScreenPos().Minus(imgui.Vec2{X: 0, Y: 1})
	dl := imgui.WindowDrawList()

	width := imgui.CalcTextSize(" X", false, 0.0).X
	height := width * 1.55
	radius := width * 0.50
	t := float32(0.0)
	if v {
		t = 1.0
	}

	r := false

	imgui.InvisibleButtonV(id, imgui.Vec2{width, height}, imgui.ButtonFlagsMouseButtonLeft)
	if imgui.IsItemClicked() {
		r = true
	}

	dl.AddRectFilledV(p, imgui.Vec2{p.X + width, p.Y + height}, bg, radius, imgui.DrawCornerFlagsAll)
	dl.AddCircleFilled(imgui.Vec2{p.X + radius, p.Y + radius + t*(width*0.5)},
		radius-1.5, imgui.PackedColorFromVec4(imgui.Vec4{1.0, 1.0, 1.0, 1.0}))

	return r
}

// when displaying imguiColorButton() or imguiBooleanButton() where the result
// is discarded, use this value with imgui.PushStyleVarFloat() with style
// imgui.StyleVarFrameRounding to indicate that the button is "read-only"
const readOnlyButtonRounding = 5.0

// imguiColourButton adds a button of a single colour.
func imguiColourButton(col imgui.Vec4, text string, dim imgui.Vec2) bool {
	imgui.PushStyleColor(imgui.StyleColorButton, col)
	imgui.PushStyleColor(imgui.StyleColorButtonHovered, col)
	imgui.PushStyleColor(imgui.StyleColorButtonActive, col)
	defer imgui.PopStyleColorV(3)
	return imgui.ButtonV(text, dim)
}

// imguiBooleanButton adds a button of either one of two colors, depending on
// the state boolean.
func imguiBooleanButton(trueCol imgui.Vec4, falseCol imgui.Vec4, state bool, text string, dim imgui.Vec2) bool {
	if state {
		imgui.PushStyleColor(imgui.StyleColorButton, trueCol)
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, trueCol)
		imgui.PushStyleColor(imgui.StyleColorButtonActive, trueCol)
	} else {
		imgui.PushStyleColor(imgui.StyleColorButton, falseCol)
		imgui.PushStyleColor(imgui.StyleColorButtonHovered, falseCol)
		imgui.PushStyleColor(imgui.StyleColorButtonActive, falseCol)
	}
	defer imgui.PopStyleColorV(3)
	return imgui.ButtonV(text, dim)
}

// imguiLabel aligns text with widget borders and positions cursor so next
// widget will follow the label. where a label parameter is required by a
// widget and you do not want it to appear, preferring the label given by
// imguiLabel(), you can use the empty string or use the double hash construct.
// For example
//
//	imgui.SliderInt("##foo", &v, s, e)
//	imguiLabel("My Slider")
func imguiLabel(text string) {
	imgui.AlignTextToFramePadding()
	imgui.Text(text)
	imgui.SameLine()
}

// imguiLabelEnd is the same imguiLabel but without the instruction to put the
// next widget on the same line.
func imguiLabelEnd(text string) {
	imgui.AlignTextToFramePadding()
	imgui.Text(text)
}

// position cursor for indented imgui.Text().
func imguiIndentText(text string) {
	p := imgui.CursorPos()
	p.X += 10
	imgui.SetCursorPos(p)
	imgui.Text(text)
}

// imguiMeasureHeight returns the height of the region drawn in the region() function.
func imguiMeasureHeight(region func()) float32 {
	p := imgui.CursorPos()
	region()
	return imgui.CursorPos().Minus(p).Y
}

// imguiMeasureWidth returns the width of the region drawn in the region()
// function.
//
// it's a bit tricky getting the width with dear imgui. it involves noting the
// current position, calling SameLine(), performing the width measurement and
// returning to the stored position.
//
// this seems to work but it caused odd results when used to measure the width
// of a table.
// func imguiMeasureWidth(region func()) float32 {
// 	p := imgui.CursorPos()
// 	region()
// 	defer imgui.SetCursorPos(imgui.CursorPos())
// 	imgui.SameLine()
// 	return imgui.CursorPos().Minus(p).X
// }

// pads imgui.Separator with additional spacing.
func imguiSeparator() {
	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()
}

// draw grid of bytes with before and after functions in addition to commit function.
func drawByteGrid(id string, data []uint8, origin uint32,
	before func(idx int), after func(idx int), commit func(idx int, value uint8)) {

	// the origin and memtop as a string
	originString := fmt.Sprintf("%08x", origin)
	memtopString := fmt.Sprintf("%08x", origin+uint32(len(data)-1))

	// find first non-matching digit of origin and memtop strings
	columnCrop := 0
	for i := 0; i < len(originString); i++ {
		if originString[i] != memtopString[i] {
			columnCrop = i
			break // for loop
		}
	}

	// the width of the row heading column
	rowHeadingWidth := len(originString) - columnCrop

	spacing := imgui.Vec2{X: 0.5, Y: 0.5}
	imgui.PushStyleVarVec2(imgui.StyleVarCellPadding, spacing)
	defer imgui.PopStyleVar()

	const numColumns = 16

	flgs := imgui.TableFlagsSizingFixedFit

	if imgui.BeginTableV(id, numColumns+1, flgs, imgui.Vec2{}, 0.0) {
		// in some situations we will return early from the drawByteGrid()
		// function so we want to make sure that EndTable() is called
		defer imgui.EndTable()

		imgui.TableSetupScrollFreeze(0, 1)

		// set up columns
		width := imguiTextWidth(rowHeadingWidth)
		imgui.TableSetupColumnV(fmt.Sprintf("%p_column0", data), imgui.TableColumnFlagsNone, width, 0)
		width = imguiTextWidth(2)
		for i := 1; i < numColumns+1; i++ {
			imgui.TableSetupColumnV(fmt.Sprintf("%p_column%d", data, i), imgui.TableColumnFlagsNone, width, 0)
		}

		// header row
		imgui.TableNextRow()

		// skip first column of the header row
		imgui.TableNextColumn()

		// try to center header with the text in the column
		leftPad := imgui.CurrentStyle().FramePadding().X

		// draw headers for each column
		for i := 0; i < numColumns; i++ {
			imgui.TableNextColumn()
			pos := imgui.CursorPos()
			pos.X += leftPad
			imgui.SetCursorPos(pos)
			imgui.Text(fmt.Sprintf("-%x", i))
		}

		// simple way of creating a gap to the main body of the table
		imgui.TableNextRow()
		imgui.TableNextRow()

		// the number of leading columns is the number of empty columns on the
		// first row
		//
		// we need to account for these leading columns when:
		// a) calculating the clipper length value
		// b) setting the idx and address values at the start of every row
		leadingColumns := int(origin % numColumns)

		// first row requires special handling in order to account for blank
		// columns on the first row
		firstRow := true

		// clipper length is divided by the number of columns and is used to
		// tell the ListClipper how much data to expect
		//
		// we add numColumns to make sure we include the last line which may be
		// an incomplete row and would otherwise be missed out of the clipper
		//
		// we also make sure we adjust for the number of leading columns
		//
		// note that this strategy requires a check that offset does not exceed
		// the actual length of the data
		clipperLen := len(data) + numColumns + leadingColumns - 1

		// offset and address will be increased as we draw each column

		var clipper imgui.ListClipper
		clipper.Begin(clipperLen / numColumns)

		for clipper.Step() {
			for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
				idx := (i * numColumns) - leadingColumns
				addr := origin + uint32(idx)

				imgui.TableNextRow()
				imgui.TableNextColumn()
				imgui.AlignTextToFramePadding()
				imgui.Text(fmt.Sprintf("%08x-", addr/16)[columnCrop+1:])

				// column limit for row changes depending on the requirements
				// of the first row
				columnLimitForRow := numColumns

				// add blank columns to first row as necessary
				if firstRow {
					for j := 0; j < leadingColumns; j++ {
						imgui.TableNextColumn()
						idx++
						addr++
					}
					columnLimitForRow -= leadingColumns
					firstRow = false
				}

				for j := 0; j < columnLimitForRow; j++ {
					// check that offset hasn't gone beyond the end of data
					if idx >= len(data) {
						break
					}

					imgui.TableNextColumn()

					if before != nil {
						before(idx)
					}

					// editable byte
					b := data[idx]

					s := fmt.Sprintf("%02x", b)
					if imguiHexInput(fmt.Sprintf("%s##%08x", id, addr), 2, &s) {
						if v, err := strconv.ParseUint(s, 16, 8); err == nil {
							commit(idx, uint8(v))
						}
					}

					if after != nil {
						after(idx)
					}

					// advance offset and addr by one
					idx++
					addr++
				}
			}
		}
	}
}

// draw grid of bytes with automated diff highlighting and tooltip handling
//
// see drawByteGrid() for more flexible alternative.
func (img *SdlImgui) drawByteGridSimple(id string, data []uint8, diff []uint8, diffCol imgui.Vec4, origin uint32, commit func(int, uint8)) {
	var a uint8
	var b uint8

	before := func(idx int) {
		// editable byte
		a = data[idx]

		// compare current RAM value with value in comparison snapshot and use
		// highlight color if it is different
		b = a
		if diff != nil {
			b = diff[idx]
		}
		if a != b {
			imgui.PushStyleColor(imgui.StyleColorFrameBg, diffCol)
		}
	}

	after := func(idx int) {
		if a != b {
			img.imguiTooltip(func() {
				imguiColorLabelSimple(fmt.Sprintf("%02x %c %02x", b, fonts.ByteChange, a), diffCol)
			}, true)
		}

		// undo any color changes
		if a != b {
			imgui.PopStyleColor()
		}
	}

	drawByteGrid(id, data, origin, before, after, commit)
}

// imguiColorLabelSimple displays a coloured icon (fonts.ColorSwatch) with a label.
// useful for generating color keys.
func imguiColorLabelSimple(label string, col imgui.Vec4) {
	imgui.BeginGroup()
	imgui.PushStyleColor(imgui.StyleColorText, col)
	imgui.Text(string(fonts.ColorSwatch))
	imgui.PopStyleColor()
	imgui.SameLine()
	imgui.Text(label)
	imgui.EndGroup()
}

// imguiColorLabel displays a coloured icon (fonts.ColorSwatch) with a label.
// unlike imguiColorLabelSimple(), the label is produced by the supplied
// function.
func imguiColorLabel(f func(), col imgui.Vec4) {
	imgui.BeginGroup()
	imgui.PushStyleColor(imgui.StyleColorText, col)
	imgui.Text(string(fonts.ColorSwatch))
	imgui.PopStyleColor()
	imgui.SameLine()
	f()
	imgui.EndGroup()
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

// use appropriate palette for television spec.
func (img *SdlImgui) imguiTVPalette() (string, packedPalette, []imgui.Vec4, []color.RGBA) {
	switch img.cache.TV.GetFrameInfo().Spec.ID {
	case "NTSC":
		return "NTSC", img.cols.packedPaletteNTSC, img.cols.paletteNTSC, specification.PaletteNTSC
	case "PAL":
		return "PAL", img.cols.packedPalettePAL, img.cols.palettePAL, specification.PalettePAL
	case "PALM":
		return "PALM", img.cols.packedPalettePAL, img.cols.palettePAL, specification.PalettePAL
	case "SECAM":
		return "SECAM", img.cols.packedPaletteSECAM, img.cols.paletteSECAM, specification.PaletteSECAM
	}
	return "unknown", img.cols.packedPaletteNTSC, img.cols.paletteNTSC, specification.PaletteNTSC
}

// draw swatch. returns true if clicked. a good response to a click event is to
// open up an instance of popupPalette.
//
// size argument should be expressed as a fraction the fraction will be applied
// to imgui.FontSize() to obtain the radius of the swatch.
func (img *SdlImgui) imguiSwatch(col uint8, size float32) (clicked bool) {
	_, pal, _, _ := img.imguiTVPalette()
	c := pal[col]

	r := imgui.FontSize() * size

	// position & dimensions of swatch
	l := imgui.FontSize() * 0.75
	p := imgui.CursorScreenPos()
	p.X += r
	p.Y += l

	// if mouse is clicked in the range of the swatch. very simple detection,
	// not accounting for the fact that the swatch is visibly circular
	if imgui.IsWindowHovered() && imgui.IsMouseClicked(0) {
		pos := imgui.MousePos()
		clicked = pos.X >= p.X-r && pos.X <= p.X+r && pos.Y >= p.Y-r && pos.Y <= p.Y+r
	}

	// draw swatch
	dl := imgui.WindowDrawList()
	dl.AddCircleFilled(p, r, c)

	// set up cursor for next widget
	p.X += 2 * r
	p.Y -= l
	imgui.SetCursorScreenPos(p)

	return clicked
}

// draw value with hex input and bit toggles. doesn't draw a label.
//
// if onWrite is nil then the hex input is not drawn.
func drawRegister(id string, val uint8, mask uint8, col imgui.PackedColor, onWrite func(uint8)) {
	if onWrite != nil {
		v := fmt.Sprintf("%02x", val)
		if imguiHexInput(id, 2, &v) {
			v, err := strconv.ParseUint(v, 16, 8)
			if err != nil {
				panic(err)
			}
			onWrite(uint8(v) & mask)
		}

		imgui.SameLine()
	}

	seq := newDrawlistSequence(imgui.Vec2{X: imgui.FrameHeight() * 0.75, Y: imgui.FrameHeight() * 0.75}, true)
	for i := 0; i < 8; i++ {
		if mask<<i&0x80 == 0x80 {
			if (val<<i)&0x80 != 0x80 {
				seq.nextItemDepressed = true
			}
			if seq.rectFill(col) {
				v := val ^ (0x80 >> i)
				if onWrite != nil {
					onWrite(uint8(v & mask))
				}
			}
		} else {
			seq.nextItemDepressed = true
			seq.rectEmpty(col)
		}

		seq.sameLine()
	}
	seq.end()
}

func drawMuteIcon(img *SdlImgui) {
	var output string

	audioMute := img.prefs.audioMuteDebugger.Get().(bool)

	if audioMute {
		output = string(fonts.AudioMute)
	} else {
		output = string(fonts.AudioUnmute)
	}

	imgui.PushStyleColor(imgui.StyleColorButton, img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonActive, img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonHovered, img.cols.Transparent)
	defer imgui.PopStyleColorV(3)

	if imgui.Button(output) {
		img.prefs.audioMuteDebugger.Set(!audioMute)
	}
}
