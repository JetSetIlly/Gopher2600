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
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

const winTimelineID = "Timeline"

type winTimeline struct {
	img  *SdlImgui
	open bool
}

func newWinTimeline(img *SdlImgui) (window, error) {
	win := &winTimeline{
		img: img,
	}
	return win, nil
}

func (win *winTimeline) init() {
}

func (win *winTimeline) id() string {
	return winTimelineID
}

func (win *winTimeline) isOpen() bool {
	return win.open
}

func (win *winTimeline) setOpen(open bool) {
	win.open = open
}

func (win *winTimeline) draw() {
	if !win.open {
		return
	}

	const winHeightRatio = 0.05
	const scanlineRatio = specification.AbsoluteMaxScanlines * winHeightRatio

	imgui.SetNextWindowPosV(imgui.Vec2{0, 0}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})

	flgs := imgui.WindowFlagsAlwaysAutoResize | imgui.WindowFlagsNoDecoration
	imgui.BeginV(win.id(), &win.open, flgs)
	defer imgui.End()

	win.drawTimeline()

	imguiSeparator()
	imgui.Text("controls")
}

func (win *winTimeline) drawTimeline() {
	const plotWidth = 2
	const plotHeight = 1

	timeline := win.img.lz.Rewind.Timeline
	dl := imgui.WindowDrawList()

	width := win.img.plt.displaySize()[0] * 0.80

	traceSize := imgui.Vec2{X: width, Y: 50}
	traceScale := (traceSize.Y - (plotHeight * 2)) / specification.AbsoluteMaxScanlines

	imgui.BeginChildV("##timelineplot", traceSize, false, imgui.WindowFlagsNone)
	pos := imgui.CursorScreenPos()

	x := pos.X
	for i := range timeline.TotalScanlines {
		l := float32(timeline.TotalScanlines[i])
		l *= traceScale

		y := pos.Y + traceSize.Y - l
		rmin := imgui.Vec2{X: x, Y: y}
		rmax := rmin.Plus(imgui.Vec2{X: plotWidth, Y: plotHeight})

		dl.AddRectFilled(rmin, rmax, win.img.cols.timelineScanlinePlot)
		x += plotWidth
	}

	imgui.EndChild()

	// indicators
	const indicatorHeight = 5

	indicatorSize := imgui.Vec2{X: width, Y: 8}
	imgui.BeginChildV("##timelineindicators", indicatorSize, false, imgui.WindowFlagsNone)
	pos = imgui.CursorScreenPos()

	// range indicator
	min := imgui.Vec2{X: pos.X + float32(timeline.AvailableStart*plotWidth), Y: pos.Y}
	max := imgui.Vec2{X: pos.X + float32(timeline.AvailableEnd*plotWidth), Y: pos.Y + indicatorHeight}
	dl.AddRectFilled(min, max, win.img.cols.timelineRewindRange)

	imgui.EndChild()

	// current frame indicator
	const currentRadius = 3
	currentSize := imgui.Vec2{X: width, Y: currentRadius}
	imgui.BeginChildV("##timelinecurrent", currentSize, false, imgui.WindowFlagsNone)
	pos = imgui.CursorScreenPos()

	fr := win.img.lz.TV.Frame
	dl.AddCircleFilled(imgui.Vec2{X: pos.X + float32(fr*plotWidth), Y: pos.Y + currentRadius}, currentRadius, win.img.cols.timelineCurrentPointer)

	imgui.EndChild()
}
