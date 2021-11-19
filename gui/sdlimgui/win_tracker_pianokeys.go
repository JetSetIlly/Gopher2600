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

const numPianoKeys = 59

func hasBlack(key int) bool {
	return (!((key-2)%7 == 0 || (key-2)%7 == 3) && key != numPianoKeys)
}

// drawlist calls to create the piano keys taken from https://github.com/shric/midi/blob/master/src/Piano.cpp
// licenced under the MIT licence
func (win *winTracker) drawPianoKeys() float32 {
	keyWidth := imgui.WindowContentRegionWidth() / float32(numPianoKeys)
	whiteKeyLength := keyWidth * 6.0
	blackKeyLength := whiteKeyLength * 0.6666

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

	return whiteKeyLength
}
