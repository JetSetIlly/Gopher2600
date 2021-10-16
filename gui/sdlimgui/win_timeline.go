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
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

const winTimelineID = "Timeline"

type winTimeline struct {
	img  *SdlImgui
	open bool

	// whether the rewind "slider" is active
	rewindingActive bool
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

	imgui.SetNextWindowPosV(imgui.Vec2{37, 732}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})

	flgs := imgui.WindowFlagsAlwaysAutoResize | imgui.WindowFlagsNoDecoration
	imgui.BeginV(win.id(), &win.open, flgs)
	defer imgui.End()

	win.drawTimeline()

	imguiSeparator()
	win.drawRewindSummary()
	imgui.SameLineV(0, 20)
	win.drawKey()
}

func (win *winTimeline) drawKey() {
	imguiColorLabel("Scanlines", win.img.cols.TimelineScanlinePlot)
	imgui.SameLineV(0, 20)
	imguiColorLabel("Left Player", win.img.cols.TimelineLeftPlayer)
	imgui.SameLineV(0, 20)
	imguiColorLabel("Rewind", win.img.cols.TimelineRewindRange)
	imgui.SameLineV(0, 20)
	imguiColorLabel("Comparison", win.img.cols.TimelineCmpPointer)
}

func (win *winTimeline) drawRewindSummary() {
	imgui.Text(fmt.Sprintf("Rewind frames: %d to %d", win.img.lz.Rewind.Timeline.AvailableStart, win.img.lz.Rewind.Timeline.AvailableEnd))
}

func (win *winTimeline) drawTimeline() {
	const plotWidth = 2
	const plotHeight = 2

	imgui.BeginGroup()
	defer imgui.EndGroup()

	timeline := win.img.lz.Rewind.Timeline
	dl := imgui.WindowDrawList()

	width := win.img.plt.displaySize()[0] * 0.80

	traceSize := imgui.Vec2{X: width, Y: 50}
	traceScale := (traceSize.Y - (plotHeight * 2)) / specification.AbsoluteMaxScanlines

	imgui.BeginChildV("##timelineplot", traceSize, false, imgui.WindowFlagsNoMove)
	pos := imgui.CursorScreenPos()

	x := pos.X
	botY := pos.Y + traceSize.Y
	for i := range timeline.FrameNum {
		sl := float32(timeline.TotalScanlines[i]) * traceScale
		rmin := imgui.Vec2{X: x, Y: botY - sl}
		rmax := rmin.Plus(imgui.Vec2{X: plotWidth, Y: plotHeight})
		dl.AddRectFilled(rmin, rmax, win.img.cols.timelineScanlinePlot)

		// left player input
		if timeline.LeftPlayerInput[i] {
			rmin := imgui.Vec2{X: x, Y: botY}
			rmax := rmin.Plus(imgui.Vec2{X: plotWidth, Y: plotHeight})
			dl.AddRectFilled(rmin, rmax, win.img.cols.timelineLeftPlayer)
		}

		// TODO: right player and panel input

		x += plotWidth
	}

	imgui.EndChild()

	// update rewind state
	if imgui.IsMouseDown(0) && (imgui.IsItemHovered() || win.rewindingActive) {
		win.rewindingActive = true
		x := imgui.MousePos().X
		x -= pos.X
		fr := int(x / plotWidth)

		if fr != win.img.lz.TV.Frame {
			s := win.img.lz.Rewind.Timeline.AvailableStart
			e := win.img.lz.Rewind.Timeline.AvailableEnd
			if fr >= s && fr <= e {
				win.img.dbg.PushRewind(fr, fr == e)
			}
		}
	} else {
		win.rewindingActive = false
	}

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

	// frame indicators
	const currentRadius = 3
	currentSize := imgui.Vec2{X: width, Y: currentRadius}
	imgui.BeginChildV("##timelinecurrent", currentSize, false, imgui.WindowFlagsNone)
	pos = imgui.CursorScreenPos()

	var fr int

	// comparison frame indicator
	if win.img.lz.Rewind.Comparison != nil {
		fr = win.img.lz.Rewind.Comparison.TV.GetState(signal.ReqFramenum)
		dl.AddCircleFilled(imgui.Vec2{X: pos.X + float32(fr*plotWidth), Y: pos.Y + currentRadius}, currentRadius, win.img.cols.timelineCmpPointer)
	}

	// current frame indicator
	fr = win.img.lz.TV.Frame
	dl.AddCircleFilled(imgui.Vec2{X: pos.X + float32(fr*plotWidth), Y: pos.Y + currentRadius}, currentRadius, win.img.cols.timelineCurrentPointer)

	imgui.EndChild()
}
