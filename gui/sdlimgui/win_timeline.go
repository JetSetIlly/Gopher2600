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

	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoDecoration)
	defer imgui.End()

	win.drawTimeline()

	imguiSeparator()
	win.drawRewindSummary()
	imgui.SameLineV(0, 20)
	win.drawKey()
}

func (win *winTimeline) drawKey() {
	imguiColorLabel("Scanlines", win.img.cols.TimelineScanlines)
	imgui.SameLineV(0, 20)
	imguiColorLabel("WSYNC", win.img.cols.TimelineWSYNC)
	if win.img.lz.CoProc.HasCoProcBus {
		imgui.SameLineV(0, 20)
		imguiColorLabel(win.img.lz.CoProc.ID, win.img.cols.TimelineCoProc)
	}
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
	const traceWidth = 2
	const traceHeight = 1
	const inputHeight = 2
	const rangeHeight = 5
	const frameIndicatorRadius = 4

	timeline := win.img.lz.Rewind.Timeline
	dl := imgui.WindowDrawList()

	var traceSize imgui.Vec2
	var pos imgui.Vec2
	var traceOffset int
	var rewindOffset int

	// the width that can be seen in the window at any one time
	availableWidth := win.img.plt.displaySize()[0] * 0.80

	// trace group
	imgui.BeginGroup()

	// traceOffset adjusts the placement of the traces in the window
	//
	// check if end of timeline overflows the available width
	if len(timeline.FrameNum)*traceWidth >= int(availableWidth) {
		traceOffset = len(timeline.FrameNum) - int(availableWidth/traceWidth)
	}

	// similar to traceOffset, rewindOffset adjusts the placement of the rewind
	// range and frame indicators (current, comparison)
	rewindOffset = traceOffset
	if len(timeline.FrameNum) > 0 {
		rewindOffset += timeline.FrameNum[0]
	}

	// scanline trace
	traceSize = imgui.Vec2{X: availableWidth, Y: 50}
	imgui.BeginChildV("##timelinescanlinetrace", traceSize, false, imgui.WindowFlagsNoMove)
	pos = imgui.CursorScreenPos()

	x := pos.X
	for i := range timeline.FrameNum[traceOffset:] {
		i += traceOffset

		// plotting from bottom
		y := pos.Y + traceSize.Y

		// scale TotalScanlines value so that it covers the entire height of traceSize
		y -= float32(timeline.TotalScanlines[i]) * traceSize.Y / specification.AbsoluteMaxScanlines

		// add jitter to trace to indicate changes in value through exaggeration
		if i > 0 {
			if timeline.TotalScanlines[i] < timeline.TotalScanlines[i-1] {
				y++
			} else if timeline.TotalScanlines[i] > timeline.TotalScanlines[i-1] {
				y--
			}
		}

		dl.AddRectFilled(imgui.Vec2{X: x, Y: y},
			imgui.Vec2{X: x + traceWidth, Y: y + traceHeight},
			win.img.cols.timelineScanlines)

		// plot WSYNC from the bottom
		y = pos.Y + traceSize.Y
		y -= float32(timeline.Counts[i].WSYNC) * traceSize.Y / specification.AbsoluteMaxClks

		// add jitter to trace to indicate changes in value through exaggeration
		if i > 0 {
			if timeline.Counts[i].WSYNC < timeline.Counts[i-1].WSYNC {
				y++
			} else if timeline.Counts[i].WSYNC > timeline.Counts[i-1].WSYNC {
				y--
			}
		}

		dl.AddRectFilled(imgui.Vec2{X: x, Y: y},
			imgui.Vec2{X: x + traceWidth, Y: y + traceHeight},
			win.img.cols.timelineWSYNC)

		// CoProc
		if win.img.lz.CoProc.HasCoProcBus {
			// plot coprocessor from the top
			y = pos.Y
			y += float32(timeline.Counts[i].CoProc) * traceSize.Y / specification.AbsoluteMaxClks

			// add jitter to trace to indicate changes in value through exaggeration
			if i > 0 {
				if timeline.Counts[i].CoProc < timeline.Counts[i-1].CoProc {
					y++
				} else if timeline.Counts[i].CoProc > timeline.Counts[i-1].CoProc {
					y--
				}
			}

			dl.AddRectFilled(imgui.Vec2{X: x, Y: y},
				imgui.Vec2{X: x + traceWidth, Y: y + traceHeight},
				win.img.cols.timelineCoProc)
		}

		x += traceWidth
	}
	imgui.EndChild()

	// input trace
	// TODO: right player and panel input
	traceSize = imgui.Vec2{X: availableWidth, Y: inputHeight}
	imgui.BeginChildV("##timelineinputtrace", traceSize, false, imgui.WindowFlagsNoMove)
	pos = imgui.CursorScreenPos()
	x = pos.X
	y := pos.Y
	for i := range timeline.FrameNum[traceOffset:] {
		i += traceOffset

		if timeline.LeftPlayerInput[i] {
			dl.AddRectFilled(imgui.Vec2{X: x, Y: y},
				imgui.Vec2{X: x + traceWidth, Y: y + inputHeight},
				win.img.cols.timelineLeftPlayer)
		}

		x += traceWidth
	}
	imgui.EndChild()

	// rewind range indicator
	traceSize = imgui.Vec2{X: availableWidth, Y: rangeHeight}
	imgui.BeginChildV("##timelineindicators", traceSize, false, imgui.WindowFlagsNoMove)
	pos = imgui.CursorScreenPos()

	dl.AddRectFilled(imgui.Vec2{X: pos.X + float32((timeline.AvailableStart-rewindOffset)*traceWidth), Y: pos.Y},
		imgui.Vec2{X: pos.X + float32((timeline.AvailableEnd-rewindOffset)*traceWidth), Y: pos.Y + traceSize.Y},
		win.img.cols.timelineRewindRange)

	imgui.EndChild()

	// frame indicators
	traceSize = imgui.Vec2{X: availableWidth, Y: frameIndicatorRadius}
	imgui.BeginChildV("##timelinecurrent", traceSize, false, imgui.WindowFlagsNoMove)
	pos = imgui.CursorScreenPos()

	// comparison frame indicator
	if win.img.lz.Rewind.Comparison != nil {
		fr := win.img.lz.Rewind.Comparison.TV.GetCoords().Frame - rewindOffset

		if fr < 0 {
			// draw triangle indicating that the comparison frame is not
			// visible on the current timline
			dl.AddTriangleFilled(imgui.Vec2{X: pos.X - frameIndicatorRadius, Y: pos.Y + frameIndicatorRadius},
				imgui.Vec2{X: pos.X + frameIndicatorRadius, Y: pos.Y + frameIndicatorRadius*2},
				imgui.Vec2{X: pos.X + frameIndicatorRadius, Y: pos.Y},
				win.img.cols.timelineCmpPointer)
		} else {
			dl.AddCircleFilled(imgui.Vec2{X: pos.X + float32(fr*traceWidth), Y: pos.Y + frameIndicatorRadius}, frameIndicatorRadius, win.img.cols.timelineCmpPointer)
		}
	}

	// current frame indicator
	fr := win.img.lz.TV.Coords.Frame - rewindOffset
	dl.AddCircleFilled(imgui.Vec2{X: pos.X + float32(fr*traceWidth), Y: pos.Y + frameIndicatorRadius}, frameIndicatorRadius, win.img.cols.timelineCurrentPointer)

	imgui.EndChild()

	imgui.EndGroup()

	// hover information for the trace ground
	hovered := imgui.IsItemHoveredV(imgui.HoveredFlagsAllowWhenOverlapped)
	hoverX := imgui.MousePos().X - pos.X

	// rewind "slider" is attached to scanline trace
	if imgui.IsMouseDown(0) && (hovered || win.rewindingActive) {
		s := win.img.lz.Rewind.Timeline.AvailableStart
		e := win.img.lz.Rewind.Timeline.AvailableEnd

		fr := int(hoverX/traceWidth) + rewindOffset
		win.rewindingActive = true

		// making sure we only call PushRewind() when we need to. also,
		// allowing mouse to travel beyond the rewind boundaries (and without
		// calling PushRewind() too often)
		if fr >= e {
			if win.img.lz.TV.Coords.Frame < e {
				win.img.dbg.RewindToFrame(fr, true)
			}
		} else if fr <= s {
			if win.img.lz.TV.Coords.Frame > s {
				win.img.dbg.RewindToFrame(fr, false)
			}
		} else if fr != win.img.lz.TV.Coords.Frame {
			win.img.dbg.RewindToFrame(fr, fr == e)
		}
	} else {
		win.rewindingActive = false

		if hovered && len(win.img.lz.Rewind.Timeline.FrameNum) > 0 {
			fr := int(hoverX/traceWidth) + traceOffset
			s := win.img.lz.Rewind.Timeline.FrameNum[0]
			e := win.img.lz.Rewind.Timeline.FrameNum[len(win.img.lz.Rewind.Timeline.FrameNum)-1]

			if fr >= s && fr <= e {
				imgui.BeginTooltip()
				imgui.Text(fmt.Sprintf("Frame: %d", fr))
				if fr >= s && fr <= e {
					// adjust text color slightly - the colors we use for the
					// plots are too dark
					textColAdj := imgui.Vec4{0.2, 0.2, 0.2, 0.0}

					imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.TimelineScanlines.Plus(textColAdj))
					imgui.Text(fmt.Sprintf("Scanlines: %d", win.img.lz.Rewind.Timeline.TotalScanlines[fr]))
					imgui.PopStyleColor()

					imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.TimelineWSYNC.Plus(textColAdj))
					imgui.Text(fmt.Sprintf("WSYNC %%: %.01f%%", win.img.lz.Rewind.Timeline.Ratios[fr].WSYNC*100))
					imgui.PopStyleColor()

					if win.img.lz.CoProc.HasCoProcBus {
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.TimelineCoProc.Plus(textColAdj))
						imgui.Text(fmt.Sprintf("%s %%: %.01f%%", win.img.lz.CoProc.ID, win.img.lz.Rewind.Timeline.Ratios[fr].CoProc*100))
						imgui.PopStyleColor()
					}
				}
				imgui.EndTooltip()
			}
		}
	}
}
