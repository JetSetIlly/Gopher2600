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
	"github.com/jetsetilly/gopher2600/hardware/peripherals/savekey/i2c"
	"github.com/jetsetilly/imgui-go/v5"
)

func drawI2C(scl i2c.Trace, sda i2c.Trace, dim imgui.Vec2, cols *imguiColors, tips tooltips) {
	if len(scl.Activity) != len(sda.Activity) {
		imgui.Text("ERROR: SCL and SDA trace lengths must be the same length")
		return
	}
	traceLength := len(scl.Activity)

	pos := imgui.CursorScreenPos()
	imgui.Dummy(dim)

	dl := imgui.WindowDrawList()
	dl.AddRectFilled(pos, pos.Plus(dim), cols.saveKeyOscBG)

	const (
		plotWidth  = float32(8)
		plotHeight = float32(2)
		xpad       = float32(0)
		ypad       = float32(2)
		gap        = float32(2)
	)

	maxActivity := min(int(dim.X/(plotWidth+gap)), traceLength)
	origin := traceLength - maxActivity

	plot := func(trace []bool, col imgui.PackedColor) {
		p := pos.Plus(imgui.Vec2{X: xpad, Y: ypad})
		for _, b := range trace[origin:] {
			level := p
			if !b {
				level = level.Plus(imgui.Vec2{Y: dim.Y - (plotHeight * 4)})
			}
			dl.AddRectFilled(level, level.Plus(imgui.Vec2{X: plotWidth, Y: plotHeight}), col)
			p = p.Plus(imgui.Vec2{X: plotWidth + gap})
		}
	}

	plot(scl.Activity, cols.saveKeyOscSCL)

	pos = pos.Plus(imgui.Vec2{Y: plotHeight * 2})
	plot(sda.Activity, cols.saveKeyOscSDA)

	tips.imguiTooltip(func() {
		x := imgui.MousePos().X - pos.X
		i := int((x-xpad)/(plotWidth+gap)) + origin
		if i > 0 && i < traceLength {
			if scl.Activity[i] {
				imguiColorLabelSimple("SCL high", cols.SaveKeyOscSCL)
			} else {
				imguiColorLabelSimple("SCL low", cols.SaveKeyOscSCL)
			}
			if sda.Activity[i] {
				imguiColorLabelSimple("SDA high", cols.SaveKeyOscSDA)
			} else {
				imguiColorLabelSimple("SDA low", cols.SaveKeyOscSDA)
			}
		}
	}, true)
}
