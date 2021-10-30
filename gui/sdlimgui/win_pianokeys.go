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
	"github.com/jetsetilly/gopher2600/tracker"
)

const winPianoKeysID = "Piano Keys"

type winPianoKeys struct {
	img  *SdlImgui
	open bool

	smallKeys bool

	blackKeys    imgui.PackedColor
	whiteKeys    imgui.PackedColor
	whiteKeysGap imgui.PackedColor
}

func newWinPianoKeys(img *SdlImgui) (window, error) {
	win := &winPianoKeys{
		img:          img,
		blackKeys:    imgui.PackedColorFromVec4(imgui.Vec4{0, 0, 0, 1.0}),
		whiteKeys:    imgui.PackedColorFromVec4(imgui.Vec4{1.0, 1.0, 0.90, 1.0}),
		whiteKeysGap: imgui.PackedColorFromVec4(imgui.Vec4{0.2, 0.2, 0.2, 1.0}),
	}
	return win, nil
}

func (win *winPianoKeys) init() {
}

func (win *winPianoKeys) id() string {
	return winPianoKeysID
}

func (win *winPianoKeys) isOpen() bool {
	return win.open
}

func (win *winPianoKeys) setOpen(open bool) {
	win.open = open
}

const (
	numPianoKeys   = 59
	keyWidth       = 15.0
	whiteKeyLength = keyWidth * 6.0
	blackKeyLength = whiteKeyLength * 0.6666
	pianoWidth     = numPianoKeys * keyWidth
)

func hasBlack(key int) bool {
	return (!((key-2)%7 == 0 || (key-2)%7 == 3) && key != numPianoKeys)
}

// drawlist calls to create the piano keys taken from https://github.com/shric/midi/blob/master/src/Piano.cpp
// licenced under the MIT licence
func (win *winPianoKeys) draw() {
	if !win.open {
		return
	}

	wp := imgui.CurrentStyle().WindowPadding()

	imgui.SetNextWindowPosV(imgui.Vec2{96, 454}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{pianoWidth + wp.X*2, whiteKeyLength + wp.Y*2}, imgui.ConditionAlways)

	imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.AudioTrackerHeader)
	imgui.PushStyleVarFloat(imgui.StyleVarWindowBorderSize, 3.0)
	imgui.PushStyleVarFloat(imgui.StyleVarWindowRounding, 3.0)
	defer imgui.PopStyleColor()
	defer imgui.PopStyleVarV(2)

	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsNoResize|imgui.WindowFlagsNoTitleBar)
	defer imgui.End()

	dl := imgui.WindowDrawList()
	p := imgui.CursorScreenPos()

	// strike color is weighted by volume
	v0 := int(win.img.lz.Tracker.LastEntry[0].Registers.Volume)
	v1 := int(win.img.lz.Tracker.LastEntry[1].Registers.Volume)
	s0 := (0.5 / float32(16-v0))
	s1 := (0.5 / float32(16-v1))
	strike0 := imgui.PackedColorFromVec4(imgui.Vec4{1.0, s0, s0, 1.0})
	strike1 := imgui.PackedColorFromVec4(imgui.Vec4{s1, 1.0, s1, 1.0})

	c0 := int(win.img.lz.Tracker.LastEntry[0].PianoKey)
	if c0 < 0 || v0 == 0 {
		c0 = tracker.NoPianoKey
	}
	c1 := int(win.img.lz.Tracker.LastEntry[1].PianoKey)
	if c1 < 0 || v1 == 0 {
		c1 = tracker.NoPianoKey
	}

	for k := 0; k < numPianoKeys; k++ {
		col := win.whiteKeys
		if k+1 == c0 {
			col = strike0
		}
		if k+1 == c1 {
			col = strike1
		}

		dl.AddRectFilledV(imgui.Vec2{p.X + float32(k)*keyWidth, p.Y},
			imgui.Vec2{p.X + float32(k)*keyWidth + keyWidth, p.Y + whiteKeyLength},
			col, 0, imgui.DrawCornerFlagsNone)
		dl.AddRectV(imgui.Vec2{p.X + float32(k)*keyWidth, p.Y},
			imgui.Vec2{p.X + float32(k)*keyWidth + keyWidth, p.Y + whiteKeyLength},
			win.whiteKeysGap, 0, imgui.DrawCornerFlagsNone, 1)
	}

	c0 = int(win.img.lz.Tracker.LastEntry[0].PianoKey)
	v0 = int(win.img.lz.Tracker.LastEntry[0].Registers.Volume)
	if c0 >= 0 || v0 == 0 {
		c0 = tracker.NoPianoKey
	} else {
		c0 *= -1
	}
	c1 = int(win.img.lz.Tracker.LastEntry[1].PianoKey)
	v1 = int(win.img.lz.Tracker.LastEntry[1].Registers.Volume)
	if c1 >= 0 || v1 == 0 {
		c1 = tracker.NoPianoKey
	} else {
		c1 *= -1
	}

	for k := 0; k < numPianoKeys; k++ {
		if hasBlack(k + 1) {
			col := win.blackKeys
			if k+1 == c0 {
				col = strike0
			}
			if k+1 == c1 {
				col = strike1
			}

			dl.AddRectFilledV(imgui.Vec2{p.X + float32(k)*keyWidth + keyWidth*3/4, p.Y},
				imgui.Vec2{p.X + float32(k)*keyWidth + keyWidth*5/4 + 1, p.Y + blackKeyLength},
				col, 0, imgui.DrawCornerFlagsNone)
			dl.AddRectV(
				imgui.Vec2{p.X + float32(k)*keyWidth + keyWidth*3/4, p.Y},
				imgui.Vec2{p.X + float32(k)*keyWidth + keyWidth*5/4 + 1, p.Y + blackKeyLength},
				win.blackKeys, 0, imgui.DrawCornerFlagsNone, 1)
		}
	}

}
