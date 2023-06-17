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
	"os"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources/unique"
	"github.com/jetsetilly/gopher2600/thumbnailer"
)

const winTimelineID = "Timeline"

type winTimeline struct {
	debuggerWin

	img *SdlImgui

	// thumbnailer will be using emulation states created in the main emulation
	// goroutine so we must thumbnail those states in the same goroutine.
	thmb          *thumbnailer.Image
	thmbTexture   uint32
	thmbFrame     int
	thmbFlipped   bool
	thmbFlippedCt int

	// mouse hover information
	isHovered  bool
	hoverX     float32
	hoverFrame int

	// is user currently scrubbing
	scrubbing bool

	// the trace offset from the previous frame
	prevTraceOffset int

	// height of toolbar
	toolbarHeight float32
}

func newWinTimeline(img *SdlImgui) (window, error) {
	win := &winTimeline{
		img: img,
	}

	var err error

	win.thmb, err = thumbnailer.NewImage(win.img.vcs.Env.Prefs)
	if err != nil {
		return nil, fmt.Errorf("debugger: %w", err)
	}

	gl.GenTextures(1, &win.thmbTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.thmbTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)

	return win, nil
}

func (win *winTimeline) init() {
}

func (win *winTimeline) id() string {
	return winTimelineID
}

const timelinePopupID = "timelinePopupID"

func (win *winTimeline) debuggerDraw() bool {
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
		return false
	}

	const winHeightRatio = 0.05
	const scanlineRatio = specification.AbsoluteMaxScanlines * winHeightRatio

	imgui.SetNextWindowPosV(imgui.Vec2{39, 722}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{875, 220}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{750, 200}, imgui.Vec2{win.img.plt.displaySize()[0] * 0.95, 300})

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsNone) {
		// trace area
		win.drawTrace()

		// toolbar
		win.toolbarHeight = imguiMeasureHeight(func() {
			imguiSeparator()
			win.drawToolbar()
		})

		// popup menu
		if imgui.BeginPopup(timelinePopupID) {
			if imgui.Selectable(fmt.Sprintf("%c Save Timeline to CSV", fonts.Disk)) {
				win.saveToCSV()
			}
			if win.isHovered {
				if imgui.Selectable(fmt.Sprintf("Set Comparison to Frame %d", win.hoverFrame)) {
					win.img.dbg.PushFunction(func() {
						win.img.dbg.Rewind.SetComparison(win.hoverFrame)
					})
				}
			}
			imgui.EndPopup()
		}
	}

	win.debuggerGeom.update()
	imgui.End()

	if win.isHovered {
		// slow the rate at which we generate thumbnails
		if win.img.polling.throttleTimelineThumbnailer() {
			win.img.dbg.PushFunction(func() {
				// thumbnailer must be run in the same goroutine as the main emulation
				win.thmb.Create(win.img.dbg.Rewind.GetState(win.hoverFrame))
			})
		}
	}

	return true
}

func (win *winTimeline) drawToolbar() {
	imgui.AlignTextToFramePadding()
	imguiColorLabelSimple("Scanlines", win.img.cols.TimelineScanlines)
	imgui.SameLine()
	imgui.AlignTextToFramePadding()
	imguiColorLabelSimple("WSYNC", win.img.cols.TimelineWSYNC)
	imgui.SameLine()
	if win.img.lz.Cart.HasCoProcBus {
		imgui.AlignTextToFramePadding()
		imguiColorLabelSimple(win.img.lz.Cart.CoProcID, win.img.cols.TimelineCoProc)
		imgui.SameLine()
	}
	imgui.AlignTextToFramePadding()
	imguiColorLabelSimple("Left Player", win.img.cols.TimelineLeftPlayer)
	imgui.SameLine()
	imgui.AlignTextToFramePadding()
	imguiColorLabelSimple("Rewind", win.img.cols.TimelineRewindRange)
	imgui.SameLine()
	imgui.AlignTextToFramePadding()
	imguiColorLabelSimple("Comparison", win.img.cols.TimelineCmpPointer)
}

func (win *winTimeline) drawRewindSummary() {
	imgui.Text(fmt.Sprintf("Rewind frames: %d to %d", win.img.lz.Rewind.Timeline.AvailableStart, win.img.lz.Rewind.Timeline.AvailableEnd))
}

func (win *winTimeline) drawTrace() {
	timeline := win.img.lz.Rewind.Timeline
	dl := imgui.WindowDrawList()

	// size of trace area elements. the size of the graph depends on the size of
	// the timeline window
	const (
		gap              = 5
		inputTrace       = 2
		rewindRangeTrace = 3
		plotWidth        = 4
		plotHeight       = 2
		cursorWidth      = 5
	)

	// the amount to allow for the icons when centering etc.
	iconRadius := win.img.glsl.fonts.defaultFontSize / 2

	// the radius of the indicator circles/triangles/etc.
	indicatorRadius := win.img.glsl.fonts.defaultFontSize / 3

	// the width that can be seen in the window at any one time. reduce by the
	// iconRadius*2 value to allow for the TV icon (current frame icon) when it
	// reaches the extreme left/right of the window
	availableWidth := imgui.ContentRegionAvail().X - iconRadius*2

	// the width of the timeline window in frames (ie. number of frames visible)
	availableWidthInFrames := int(availableWidth / plotWidth)

	// size of entire timeline trace area
	traceSize := imgui.Vec2{X: availableWidth, Y: imgui.ContentRegionAvail().Y - win.toolbarHeight}

	// height of the graph portion of the trace area
	graphHeight := traceSize.Y - (gap*2 + inputTrace + gap + rewindRangeTrace + gap + iconRadius*2 + gap)

	// check if end of timeline overflows the available width and adjust offset
	// so that the trace is right-justified (for want of a better description)
	//
	// we also don't want to decrease the traceoffset from a previous high value
	//
	// TODO: there is no horizontal scrolling mechanism for the timeline yet but
	// if we ever add one then the use of prevTraceOffset will need to be
	// refined
	var traceOffset int
	if len(timeline.FrameNum)*plotWidth >= int(availableWidth) {
		traceOffset = len(timeline.FrameNum) - availableWidthInFrames
	}
	if traceOffset > win.prevTraceOffset {
		win.prevTraceOffset = traceOffset
	} else {
		traceOffset = win.prevTraceOffset
	}

	// similar to traceOffset, rewindOffset adjusts the placement of the rewind
	// range and frame indicators
	rewindOffset := traceOffset
	if len(timeline.FrameNum) > 0 {
		rewindOffset += timeline.FrameNum[0]
	}

	// list of scanline jitter points to indicate. these will be found during
	// the plot of the scanline trace and then used to draw the jitter
	// indicators in a second loop
	var scanlineJitter []int
	scanlineJitter = append(scanlineJitter, 0)

	// scanline/coproc/WSYNC trace
	imgui.BeginChildV("##timelinetrace", traceSize, false, imgui.WindowFlagsNoMove)

	// the position of the trace widget. move right slightly to create a margin
	// of width iconRadius
	rootPos := imgui.CursorScreenPos()
	rootPos.X += iconRadius
	imgui.SetCursorScreenPos(rootPos)

	// the Y position of each trace area
	yPos := rootPos.Y + gap

	// rewind start/end X positions
	rewindStartX := rootPos.X + float32((timeline.AvailableStart-rewindOffset)*plotWidth)
	if rewindStartX < rootPos.X {
		rewindStartX = rootPos.X
	}
	rewindEndX := rootPos.X + float32((timeline.AvailableEnd-rewindOffset)*plotWidth)

	// draw frame guides
	const guideFrameCount = 20
	imgui.PushFont(win.img.glsl.fonts.diagram)

	var guideStart int
	if len(timeline.FrameNum) > 0 {
		guideStart = timeline.FrameNum[traceOffset]
	}

	guideX := rootPos.X
	for fn := guideStart; fn < guideStart+availableWidthInFrames; fn++ {
		if fn%guideFrameCount == 0 {
			// draw vertical frame guides
			top := imgui.Vec2{X: guideX, Y: rootPos.Y}
			bot := imgui.Vec2{X: guideX, Y: rootPos.Y + traceSize.Y}
			dl.AddRectFilled(top, bot, win.img.cols.timelineGuides)

			// label frame guides with frame numbers
			bot.X += 5
			bot.Y -= win.img.glsl.fonts.diagramSize / 2
			dl.AddText(bot, win.img.cols.timelineGuidesLabel, fmt.Sprintf("%d", fn))
		}
		guideX += plotWidth
	}
	imgui.PopFont()

	// show cursor
	if win.isHovered {
		dl.AddRectFilled(imgui.Vec2{X: win.hoverX - cursorWidth/2, Y: rootPos.Y},
			imgui.Vec2{X: win.hoverX + cursorWidth/2, Y: rootPos.Y + traceSize.Y},
			win.img.cols.timelineHoverCursor)

		// show thumbnail alongside cursor
		if win.img.prefs.showTimelineThumbnail.Get().(bool) {
			thumbnailSize := imgui.Vec2{X: specification.ClksVisible * 3, Y: specification.AbsoluteMaxScanlines}
			thumbnailSize = thumbnailSize.Times(traceSize.Y / specification.AbsoluteMaxScanlines)

			// position thumbnail before or after the cursor depending on flip value
			// and the proxity of the thumbnail to the timeline boundary
			//
			// thumbFlippedCt gives a half-second grace before flip happens.
			// this looks and feels better
			var thumbnailPos imgui.Vec2
			if win.thmbFlipped {
				thumbnailPos = imgui.Vec2{X: win.hoverX + cursorWidth*2, Y: rootPos.Y}
				if thumbnailPos.X+thumbnailSize.X > rootPos.X+traceSize.X {
					win.thmbFlippedCt++
					if win.thmbFlippedCt > int(win.img.plt.mode.RefreshRate/2) {
						win.thmbFlipped = false
						thumbnailPos.X = win.hoverX - cursorWidth*2 - thumbnailSize.X
					}
				} else {
					win.thmbFlippedCt = 0
				}
			} else {
				thumbnailPos = imgui.Vec2{X: win.hoverX - cursorWidth*2 - thumbnailSize.X, Y: rootPos.Y}
				if thumbnailPos.X < rootPos.X {
					win.thmbFlippedCt++
					if win.thmbFlippedCt > int(win.img.plt.mode.RefreshRate/2) {
						win.thmbFlipped = true
						thumbnailPos.X = win.hoverX + cursorWidth*2
					}
				} else {
					win.thmbFlippedCt = 0
				}
			}
			imgui.SetCursorScreenPos(thumbnailPos)

			imgui.ImageV(imgui.TextureID(win.thmbTexture), thumbnailSize,
				imgui.Vec2{}, imgui.Vec2{1, 1},
				win.img.cols.TimelineThumbnailTint, imgui.Vec4{})

			imgui.SetCursorScreenPos(rootPos)
		}
	}

	// draw main trace plot
	plotX := rootPos.X
	for i := range timeline.FrameNum[traceOffset:] {
		// adjust index by starting point
		i += traceOffset

		// SCANLINE TRACE
		plotY := yPos + graphHeight

		// scale TotalScanlines value so that it covers the entire height of traceSize
		plotY -= float32(timeline.TotalScanlines[i]) * graphHeight / specification.AbsoluteMaxScanlines

		// add jitter to trace to indicate changes in value through exaggeration
		if i > 0 {
			if timeline.TotalScanlines[i] != timeline.TotalScanlines[i-1] {
				if timeline.TotalScanlines[i] < timeline.TotalScanlines[i-1] {
					plotY++
				} else if timeline.TotalScanlines[i] > timeline.TotalScanlines[i-1] {
					plotY--
				}

				// add to jitter history if it hasn't been updated for a while
				prev := scanlineJitter[len(scanlineJitter)-1]
				j := i - traceOffset
				if j-prev > 3 {
					scanlineJitter = append(scanlineJitter, j)
				}
			}
		}

		// draw scanline trace
		dl.AddRectFilled(imgui.Vec2{X: plotX, Y: plotY},
			imgui.Vec2{X: plotX + plotWidth, Y: plotY + plotHeight},
			win.img.cols.timelineScanlines)

		// WSYNC TRACE

		// plot WSYNC from the bottom
		plotY = yPos + graphHeight
		plotY -= float32(timeline.Counts[i].WSYNC) * graphHeight / specification.AbsoluteMaxClks

		// add jitter to trace to indicate changes in value through exaggeration
		if i > 0 {
			if timeline.Counts[i].WSYNC < timeline.Counts[i-1].WSYNC {
				plotY++
			} else if timeline.Counts[i].WSYNC > timeline.Counts[i-1].WSYNC {
				plotY--
			}
		}

		// plot a dotted line if count isn't valid and a solid line if it is
		dl.AddRectFilled(imgui.Vec2{X: plotX, Y: plotY},
			imgui.Vec2{X: plotX + plotWidth, Y: plotY + plotHeight},
			win.img.cols.timelineWSYNC)

		// COPROCESSOR TRACE

		// plot coprocessor from the top
		if win.img.lz.Cart.HasCoProcBus {
			plotY = yPos
			plotY += float32(timeline.Counts[i].CoProc) * graphHeight / specification.AbsoluteMaxClks

			// add jitter to trace to indicate changes in value through exaggeration
			if i > 0 {
				if timeline.Counts[i].CoProc < timeline.Counts[i-1].CoProc {
					plotY++
				} else if timeline.Counts[i].CoProc > timeline.Counts[i-1].CoProc {
					plotY--
				}
			}

			// plot a dotted line if count isn't valid and a solid line if it is
			dl.AddRectFilled(imgui.Vec2{X: plotX, Y: plotY},
				imgui.Vec2{X: plotX + plotWidth, Y: plotY + plotHeight},
				win.img.cols.timelineCoProc)
		}

		plotX += plotWidth
	}
	yPos += graphHeight + gap

	// input trace
	// TODO: right player and panel input
	plotX = rootPos.X
	for i := range timeline.FrameNum[traceOffset:] {
		i += traceOffset
		if timeline.LeftPlayerInput[i] {
			dl.AddRectFilled(imgui.Vec2{X: plotX, Y: yPos},
				imgui.Vec2{X: plotX + plotWidth, Y: yPos + inputTrace},
				win.img.cols.timelineLeftPlayer)
		}
		plotX += plotWidth
	}
	yPos += inputTrace + gap

	// rewind range indicator
	dl.AddRectFilled(imgui.Vec2{X: rewindStartX, Y: yPos},
		imgui.Vec2{X: rewindEndX, Y: yPos + rewindRangeTrace},
		win.img.cols.timelineRewindRange)
	yPos += rewindRangeTrace + gap

	// jitter indicators
	for _, i := range scanlineJitter[1:] {
		dl.AddTriangleFilled(
			imgui.Vec2{
				X: rootPos.X + float32(i*plotWidth) - indicatorRadius,
				Y: yPos + indicatorRadius*2,
			},
			imgui.Vec2{
				X: rootPos.X + float32(i*plotWidth),
				Y: yPos,
			},
			imgui.Vec2{
				X: rootPos.X + float32(i*plotWidth) + indicatorRadius,
				Y: yPos + indicatorRadius*2,
			},
			win.img.cols.timelineScanlines)
	}

	// comparison frame indicator
	if win.img.lz.Rewind.Comparison != nil && len(timeline.FrameNum) > 0 {
		fr := win.img.lz.Rewind.Comparison.TV.GetCoords().Frame - rewindOffset

		if fr < 0 {
			// draw triangle indicating that the comparison frame is not visible
			dl.AddTriangleFilled(imgui.Vec2{X: rootPos.X - indicatorRadius, Y: yPos + indicatorRadius},
				imgui.Vec2{X: rootPos.X + indicatorRadius, Y: yPos + indicatorRadius*2},
				imgui.Vec2{X: rootPos.X + indicatorRadius, Y: yPos},
				win.img.cols.timelineCmpPointer)
		} else {
			dl.AddCircleFilled(imgui.Vec2{X: rootPos.X + float32(fr*plotWidth), Y: yPos + indicatorRadius}, indicatorRadius, win.img.cols.timelineCmpPointer)
		}
	}

	// current frame indicator
	fr := win.img.lz.TV.Coords.Frame - rewindOffset
	if fr < 0 {
		// draw triangle indicating that the current frame is not visible
		// if the comparison frame indicator is also not visible then this
		// triangle (the current frame triangle) supercedes it
		dl.AddTriangleFilled(imgui.Vec2{X: rootPos.X - indicatorRadius, Y: yPos + indicatorRadius},
			imgui.Vec2{X: rootPos.X + indicatorRadius, Y: yPos + indicatorRadius*2},
			imgui.Vec2{X: rootPos.X + indicatorRadius, Y: yPos},
			win.img.cols.timelineRewindRange)
	}
	dl.AddText(imgui.Vec2{X: rootPos.X + float32(fr*plotWidth) - iconRadius, Y: yPos},
		win.img.cols.timelineRewindRange, string(fonts.TV))

	// end of trace area
	imgui.EndChild()

	// no mouse handling for timeline window if popup window is open
	if imgui.IsPopupOpen(timelinePopupID) {
		return
	}

	// check for mouse hover over rewindable area
	mouse := imgui.MousePos()
	win.isHovered = mouse.X >= rewindStartX && mouse.X <= rewindEndX &&
		mouse.Y >= rootPos.Y && mouse.Y <= rootPos.Y+traceSize.Y
	win.hoverX = mouse.X

	// rewind support
	rewindX := win.hoverX - rootPos.X
	rewindStartFrame := win.img.lz.Rewind.Timeline.AvailableStart
	rewindEndFrame := win.img.lz.Rewind.Timeline.AvailableEnd

	// frame number for hover position
	win.hoverFrame = int(rewindX/plotWidth) + rewindOffset

	// scrub detection works by looking for the initial click (IsMouseClicked()
	// function) in the rewind area
	win.scrubbing = (win.scrubbing && imgui.IsMouseDown(0)) ||
		(imgui.IsMouseClicked(0) && (win.isHovered || win.scrubbing))

	// mouse handling
	if imgui.IsMouseDown(1) && imgui.IsItemHovered() {
		imgui.OpenPopup(timelinePopupID)

	} else if win.scrubbing {
		// making sure we only call PushRewind() when we need to. also,
		// allowing mouse to travel beyond the rewind boundaries (and without
		// calling PushRewind() too often)
		if win.hoverFrame >= rewindEndFrame {
			if win.img.lz.TV.Coords.Frame < rewindEndFrame {
				win.img.dbg.RewindToFrame(win.hoverFrame, true)
			}
		} else if win.hoverFrame <= rewindStartFrame {
			if win.img.lz.TV.Coords.Frame > rewindStartFrame {
				win.img.dbg.RewindToFrame(win.hoverFrame, false)
			}
		} else if win.hoverFrame != win.img.lz.TV.Coords.Frame {
			win.img.dbg.RewindToFrame(win.hoverFrame, win.hoverFrame == rewindEndFrame)
		}
	} else {
		if imgui.IsItemHovered() && len(win.img.lz.Rewind.Timeline.FrameNum) > 0 {
			traceHoverIdx := int(rewindX/plotWidth) + traceOffset
			traceStartFrame := win.img.lz.Rewind.Timeline.FrameNum[0]
			traceEndFrame := win.img.lz.Rewind.Timeline.FrameNum[len(win.img.lz.Rewind.Timeline.FrameNum)-1]
			traceHoverFrame := traceHoverIdx + traceStartFrame

			if traceHoverFrame >= traceStartFrame && traceHoverFrame <= traceEndFrame {
				win.img.imguiTooltip(func() {
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
					imgui.Text(fmt.Sprintf("%.02f%%", win.img.lz.Rewind.Timeline.Ratios[traceHoverIdx].WSYNC*100))
					imgui.PopStyleColor()

					if win.img.lz.Cart.HasCoProcBus {
						imgui.Text(win.img.lz.Cart.CoProcID)
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.TimelineCoProc.Plus(textColAdj))
						imgui.SameLine()
						imgui.Text(fmt.Sprintf("%.02f%%", win.img.lz.Rewind.Timeline.Ratios[traceHoverIdx].CoProc*100))
						imgui.PopStyleColor()
					}
				}, false)
			}
		}
	}
}

func (win *winTimeline) saveToCSV() {
	// open unique file
	fn := unique.Filename("timeline", win.img.lz.Cart.Shortname)
	fn = fmt.Sprintf("%s.csv", fn)
	f, err := os.Create(fn)
	if err != nil {
		logger.Logf("sdlimgui", "could not save timeline CSV: %v", err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Logf("sdlimgui", "error saving timeline CSV: %v", err)
		}
	}()

	f.WriteString("Frame Num,")
	f.WriteString("Scanlines,")
	f.WriteString("CoProc,")
	f.WriteString("WSYNC,")
	f.WriteString("Left Player,")
	f.WriteString("Right Player,")
	f.WriteString("Panel")
	f.WriteString("\n")

	timeline := win.img.lz.Rewind.Timeline
	for i, n := range timeline.FrameNum {
		f.WriteString(fmt.Sprintf("%d,", n))
		f.WriteString(fmt.Sprintf("%d,", timeline.TotalScanlines[i]))
		f.WriteString(fmt.Sprintf("%d,", timeline.Counts[i].CoProc))
		f.WriteString(fmt.Sprintf("%d,", timeline.Counts[i].WSYNC))
		f.WriteString(fmt.Sprintf("%v,", timeline.LeftPlayerInput[i]))
		f.WriteString(fmt.Sprintf("%v,", timeline.RightPlayerInput[i]))
		f.WriteString(fmt.Sprintf("%v,", timeline.PanelInput[i]))
		f.WriteString("\n")
	}
}
