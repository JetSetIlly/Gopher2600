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
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/imgui-go/v5"
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

// draw vertical toggle button at current cursor position. returns true if toggle has been clicked.
func imguiToggleButton(id string, v *bool, fg imgui.Vec4, bg imgui.Vec4, vertical bool, scaling float32) bool {
	bgCol := imgui.PackedColorFromVec4(bg)
	fgCol := imgui.PackedColorFromVec4(fg)
	p := imgui.CursorScreenPos()
	dl := imgui.WindowDrawList()

	if vertical {
		width := imgui.FrameHeight() * scaling
		height := width * 1.55
		radius := width * 0.3
		positioning := width * 0.50
		t := float32(0.0)
		if *v {
			t = 1.0
		}

		r := false

		imgui.InvisibleButtonV(id, imgui.Vec2{X: width, Y: height}, imgui.ButtonFlagsMouseButtonLeft)
		if imgui.IsItemClicked() {
			*v = !(*v)
			r = true
		}

		dl.AddRectFilledV(p, imgui.Vec2{X: p.X + width, Y: p.Y + height}, bgCol, positioning, imgui.DrawCornerFlagsAll)
		dl.AddCircleFilled(imgui.Vec2{X: p.X + positioning, Y: p.Y + positioning + t*(width*0.5)},
			radius, fgCol)

		return r
	}

	// horizontal

	height := imgui.FrameHeight() * scaling
	width := height * 1.55
	radius := height * 0.3
	positioning := height * 0.50
	t := float32(0.0)
	if *v {
		t = 1.0
	}

	var clicked bool
	imgui.InvisibleButtonV(id, imgui.Vec2{X: width, Y: height}, imgui.ButtonFlagsMouseButtonLeft)
	if imgui.IsItemClicked() {
		*v = !(*v)
		clicked = true
	}

	p = p.Plus(imgui.Vec2{X: 0, Y: imgui.FrameHeight() * ((1.0 - scaling) * 0.5)})

	dl.AddRectFilledV(p, imgui.Vec2{X: p.X + width, Y: p.Y + height}, bgCol, positioning, imgui.DrawCornerFlagsAll)
	dl.AddCircleFilled(imgui.Vec2{X: p.X + positioning + t*(height*0.5), Y: p.Y + positioning},
		radius, fgCol)

	return clicked
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

// pads imgui.Separator with additional spacing.
func imguiSeparator() {
	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()
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

// draw swatch. returns true if clicked. a good response to a click event is to
// open up an instance of popupPalette.
//
// size argument should be expressed as a fraction the fraction will be applied
// to imgui.FontSize() to obtain the radius of the swatch.
func (img *SdlImgui) imguiTVColourSwatch(col uint8, size float32) (clicked bool) {
	ok, _, _ := img.imguiTVColourSwatchWithGeom(col, size, false)
	return ok
}

// like imguiTVColourSwatch but returns the centre point and radius, in addition to the clicked result
func (img *SdlImgui) imguiTVColourSwatchWithGeom(col uint8, size float32, highlight bool) (clicked bool, centre imgui.Vec2, radius float32) {
	r := imgui.FontSize() * size

	// position & dimensions of swatch
	l := imgui.FontSize() * size
	p := imgui.CursorScreenPos()
	p = p.Plus(imgui.Vec2{X: r, Y: l})

	// if mouse is clicked in the range of the swatch. very simple detection,
	// not accounting for the fact that the swatch is visibly circular
	if imgui.IsWindowHovered() && imgui.IsMouseClicked(0) {
		pos := imgui.MousePos()
		clicked = pos.X >= p.X-r && pos.X <= p.X+r && pos.Y >= p.Y-r && pos.Y <= p.Y+r
	}

	// draw swatch
	dl := imgui.WindowDrawList()
	if highlight {
		dl.AddCircleV(p, r, img.getTVColour(col), 0, 3.0)
		dl.AddCircleFilled(p, r*0.65, img.getTVColour(col))
	} else {
		dl.AddCircleFilled(p, r, img.getTVColour(col))
	}

	// set up cursor for next widget
	imgui.SetCursorScreenPos(p.Plus(imgui.Vec2{X: r, Y: -l}))
	imgui.Dummy(imgui.Vec2{X: 0, Y: r * 2})

	return clicked, p, r
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
	for i := range 8 {
		if mask<<i&0x80 == 0x80 {
			if (val<<i)&0x80 != 0x80 {
				seq.nextItemDepressed = true
			}
			if seq.rectFill(col) {
				v := val ^ (0x80 >> i)
				if onWrite != nil {
					onWrite(v & mask)
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

// draw a group of imgui widgets by calling the supplied function. if the
// disabled flag is enabled then the widgets will be disabled and ghosted
func drawDisabled(disabled bool, f func()) {
	if !disabled {
		f()
		return
	}
	imgui.BeginDisabled()
	f()
	imgui.EndDisabled()
}

// like drawDisabled() but the widget is invisible
func drawInvisible(invisble bool, f func()) {
	if !invisble {
		f()
		return
	}
	imgui.BeginDisabled()
	imgui.PushStyleVarFloat(imgui.StyleVarAlpha, 0.0)
	f()
	imgui.PopStyleVar()
	imgui.EndDisabled()
}
