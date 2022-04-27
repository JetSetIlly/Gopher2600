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

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/thumbnailer"
)

const winTimelineID = "Timeline"

type winTimeline struct {
	debuggerWin

	img *SdlImgui

	// whether the rewind "slider" is active
	rewindingActive bool

	// thumbnailer will be using emulation states created in the main emulation
	// goroutine so we must thumbnail those states in the same goroutine.
	thmb        *thumbnailer.Thumbnailer
	thmbTexture uint32
	thmbFrame   int
}

func newWinTimeline(img *SdlImgui) (window, error) {
	win := &winTimeline{
		img: img,
	}

	var err error

	win.thmb, err = thumbnailer.NewThumbnailer(win.img.vcs.Instance.Prefs)
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}

	gl.GenTextures(1, &win.thmbTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.thmbTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)

	return win, nil
}

func (win *winTimeline) init() {
}

func (win *winTimeline) id() string {
	return winTimelineID
}

func (win *winTimeline) debuggerDraw() {
	// receive new thumbnail data and copy to texture
	select {
	case img := <-win.thmb.Render:
		if img != nil {
			gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(img.Stride)/4)
			defer gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)

			gl.BindTexture(gl.TEXTURE_2D, win.thmbTexture)
			gl.TexImage2D(gl.TEXTURE_2D, 0,
				gl.RGBA, int32(img.Bounds().Size().X), int32(img.Bounds().Size().Y), 0,
				gl.RGBA, gl.UNSIGNED_BYTE,
				gl.Ptr(img.Pix))
		}
	default:
	}

	if !win.debuggerOpen {
		return
	}

	const winHeightRatio = 0.05
	const scanlineRatio = specification.AbsoluteMaxScanlines * winHeightRatio

	imgui.SetNextWindowPosV(imgui.Vec2{39, 722}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.drawTimeline()
		imguiSeparator()
		win.drawKey()
	}

	imgui.End()
}

const (
	traceWidth           = 2
	traceHeight          = 1
	inputHeight          = 2
	rangeHeight          = 5
	frameIndicatorRadius = 4
	unmeasuredDotPitch   = traceWidth + 1
)

func (win *winTimeline) drawKey() {
	imguiColorLabel("Scanlines", win.img.cols.TimelineScanlines)
	imgui.SameLine()
	imguiColorLabel("WSYNC", win.img.cols.TimelineWSYNC)
	imgui.SameLine()
	if win.img.lz.Cart.HasCoProcBus {
		imguiColorLabel(win.img.lz.Cart.CoProcID, win.img.cols.TimelineCoProc)
		imgui.SameLine()
	}
	imguiColorLabel("Left Player", win.img.cols.TimelineLeftPlayer)
	imgui.SameLine()
	imguiColorLabel("Rewind", win.img.cols.TimelineRewindRange)
	imgui.SameLine()
	imguiColorLabel("Comparison", win.img.cols.TimelineCmpPointer)
}

func (win *winTimeline) drawRewindSummary() {
	imgui.Text(fmt.Sprintf("Rewind frames: %d to %d", win.img.lz.Rewind.Timeline.AvailableStart, win.img.lz.Rewind.Timeline.AvailableEnd))
}

func (win *winTimeline) drawTimeline() {
	timeline := win.img.lz.Rewind.Timeline
	dl := imgui.WindowDrawList()

	var traceSize imgui.Vec2
	var pos imgui.Vec2
	var traceOffset int
	var rewindOffset int

	// the width that can be seen in the window at any one time
	availableWidth := win.img.plt.displaySize()[0] * 0.80

	// whether the timeline is hovered over. each child in the trace group is
	// tested and the results ORed together
	hovered := false

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
	imgui.BeginChildV("##timelinetrace", traceSize, false, imgui.WindowFlagsNoMove)
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

		// plottin timeline counts only if the counts entry is valid

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

		// plot a dotted line if count isn't valid and a solid line if it is
		dl.AddRectFilled(imgui.Vec2{X: x, Y: y},
			imgui.Vec2{X: x + traceWidth, Y: y + traceHeight},
			win.img.cols.timelineWSYNC)

		// plot coprocessor from the top
		if win.img.lz.Cart.HasCoProcBus {
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

			// plot a dotted line if count isn't valid and a solid line if it is
			dl.AddRectFilled(imgui.Vec2{X: x, Y: y},
				imgui.Vec2{X: x + traceWidth, Y: y + traceHeight},
				win.img.cols.timelineCoProc)
		}

		x += traceWidth
	}
	imgui.EndChild()
	hovered = hovered || imgui.IsItemHovered()

	// input trace
	// TODO: right player and panel input
	traceSize = imgui.Vec2{X: availableWidth, Y: inputHeight}
	imgui.BeginChildV("##timelinetrace_input", traceSize, false, imgui.WindowFlagsNoMove)
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
	hovered = hovered || imgui.IsItemHovered()

	// rewind range indicator
	traceSize = imgui.Vec2{X: availableWidth, Y: rangeHeight}
	imgui.BeginChildV("##timelinetrace_indicators", traceSize, false, imgui.WindowFlagsNoMove)
	pos = imgui.CursorScreenPos()

	dl.AddRectFilled(imgui.Vec2{X: pos.X + float32((timeline.AvailableStart-rewindOffset)*traceWidth), Y: pos.Y},
		imgui.Vec2{X: pos.X + float32((timeline.AvailableEnd-rewindOffset)*traceWidth), Y: pos.Y + traceSize.Y},
		win.img.cols.timelineRewindRange)

	imgui.EndChild()
	hovered = hovered || imgui.IsItemHovered()

	// frame indicators
	traceSize = imgui.Vec2{X: availableWidth, Y: frameIndicatorRadius}
	imgui.BeginChildV("##timelinetrace_current", traceSize, false, imgui.WindowFlagsNoMove)
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
	hovered = hovered || imgui.IsItemHovered()

	imgui.EndGroup()

	// mouse hover position
	hoverX := imgui.MousePos().X - pos.X

	rewindStartFrame := win.img.lz.Rewind.Timeline.AvailableStart
	rewindEndFrame := win.img.lz.Rewind.Timeline.AvailableEnd
	rewindHoverFrame := int(hoverX/traceWidth) + rewindOffset

	// hover and clicking works on the group

	if imgui.IsMouseDown(0) && (hovered || win.rewindingActive) {
		win.rewindingActive = true

		// making sure we only call PushRewind() when we need to. also,
		// allowing mouse to travel beyond the rewind boundaries (and without
		// calling PushRewind() too often)
		if rewindHoverFrame >= rewindEndFrame {
			if win.img.lz.TV.Coords.Frame < rewindEndFrame {
				win.img.dbg.RewindToFrame(rewindHoverFrame, true)
			}
		} else if rewindHoverFrame <= rewindStartFrame {
			if win.img.lz.TV.Coords.Frame > rewindStartFrame {
				win.img.dbg.RewindToFrame(rewindHoverFrame, false)
			}
		} else if rewindHoverFrame != win.img.lz.TV.Coords.Frame {
			win.img.dbg.RewindToFrame(rewindHoverFrame, rewindHoverFrame == rewindEndFrame)
		}
	} else {
		win.rewindingActive = false

		if hovered && len(win.img.lz.Rewind.Timeline.FrameNum) > 0 {
			traceHoverIdx := int(hoverX/traceWidth) + traceOffset
			traceStartFrame := win.img.lz.Rewind.Timeline.FrameNum[0]
			traceEndFrame := win.img.lz.Rewind.Timeline.FrameNum[len(win.img.lz.Rewind.Timeline.FrameNum)-1]
			traceHoverFrame := traceHoverIdx + traceStartFrame

			if traceHoverFrame >= traceStartFrame && traceHoverFrame <= traceEndFrame {
				thumbnail := rewindHoverFrame >= rewindStartFrame && rewindHoverFrame <= rewindEndFrame

				imguiTooltip(func() {
					flgs := imgui.TableFlagsNone
					if thumbnail {
						flgs = imgui.TableFlagsPadOuterX
					}
					if imgui.BeginTableV("timelineTooltip", 2, flgs, imgui.Vec2{}, 10.0) {
						imgui.TableNextRow()
						imgui.TableNextColumn()

						imgui.Text(fmt.Sprintf("Frame: %d", traceHoverFrame))

						// adjust text color slightly - the colors we use for the
						// plots are too dark
						textColAdj := imgui.Vec4{0.2, 0.2, 0.2, 0.0}

						imgui.Text("Scanlines:")
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.TimelineScanlines.Plus(textColAdj))
						imgui.SameLine()
						imgui.Text(fmt.Sprintf("%d", win.img.lz.Rewind.Timeline.TotalScanlines[traceHoverIdx]))
						imgui.PopStyleColor()

						imgui.Text("WSYNC:")
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.TimelineWSYNC.Plus(textColAdj))
						imgui.SameLine()
						imgui.Text(fmt.Sprintf("%.01f%%", win.img.lz.Rewind.Timeline.Ratios[traceHoverIdx].WSYNC*100))
						imgui.PopStyleColor()

						if win.img.lz.Cart.HasCoProcBus {
							imgui.Text(win.img.lz.Cart.CoProcID)
							imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.TimelineCoProc.Plus(textColAdj))
							imgui.SameLine()
							imgui.Text(fmt.Sprintf("%.01f%%", win.img.lz.Rewind.Timeline.Ratios[traceHoverIdx].CoProc*100))
							imgui.PopStyleColor()
						}

						// show rewind thumbnail if hover is in range of rewind
						if thumbnail {
							imgui.TableNextColumn()

							// selecting the correct thumbnail requires different indexing than the timline
							if win.thmbFrame != rewindHoverFrame {
								win.thmbFrame = rewindHoverFrame

								// slow the rate at which we generate thumbnails
								if win.img.polling.timelineThumbnailerWait() {
									win.img.dbg.PushRawEvent(func() {
										// thumbnailer must be run in the same goroutine as the main emulation
										win.thmb.SingleFrameFromRewindState(win.img.dbg.Rewind.GetState(rewindHoverFrame))
									})
								}
							}

							imgui.Image(imgui.TextureID(win.thmbTexture), imgui.Vec2{specification.ClksVisible * 3, specification.AbsoluteMaxScanlines}.Times(0.3))
						}

						imgui.EndTable()
					}
				}, false)
			}
		}
	}
}
