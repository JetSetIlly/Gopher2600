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

	// whether the rewind "slider" is active
	rewindingActive bool

	// thumbnailer will be using emulation states created in the main emulation
	// goroutine so we must thumbnail those states in the same goroutine.
	thmb          *thumbnailer.Image
	thmbTexture   uint32
	thmbFrame     int
	thmbFlipped   bool
	thmbFlippedCt int

	// mouse hover information
	isHoveredInRewindArea bool
	hoverX                float32

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
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{750, 200}, imgui.Vec2{win.img.plt.displaySize()[0] * 0.95, 300})

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.drawTrace()
		win.toolbarHeight = imguiMeasureHeight(func() {
			imguiSeparator()
			win.drawKey()
		})
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winTimeline) drawKey() {
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

	// size of trace areas
	const (
		traceGap         = 5
		traceRewind      = 3
		traceInput       = 2
		traceFrame       = 4
		tracePlotWidth   = 4
		tracePlotHeight  = 2
		traceCursorWidth = 5
	)

	const traceHeight = traceGap + traceGap + traceRewind + traceGap + traceInput + traceGap + traceFrame*4
	traceMain := (imgui.ContentRegionAvail().Y - win.toolbarHeight - traceHeight)

	// the width that can be seen in the window at any one time
	// displayWidth := win.img.plt.displaySize()[0] * 0.95
	displayWidth := imgui.ContentRegionAvail().X

	// the width of the timeline window in frames (ie. number of frames visible)
	displayWidthInFrames := int(displayWidth / tracePlotWidth)

	// size of entire timeline trace area
	traceSize := imgui.Vec2{X: displayWidth, Y: traceHeight + traceMain}

	// check if end of timeline overflows the available width and adjust offset
	// so that the trace is right-justified (for want of a better description)
	var traceOffset int
	if len(timeline.FrameNum)*tracePlotWidth >= int(displayWidth) {
		traceOffset = len(timeline.FrameNum) - displayWidthInFrames
	}
	traceFrameMax := traceOffset + displayWidthInFrames

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

	// the position of the trace widget
	rootPos := imgui.CursorScreenPos()

	// the Y position of each trace area
	yPos := rootPos.Y + traceGap

	// rewind start/end X positions
	rewindStartX := rootPos.X + float32((timeline.AvailableStart-rewindOffset)*tracePlotWidth)
	rewindEndX := rootPos.X + float32((timeline.AvailableEnd-rewindOffset)*tracePlotWidth)

	// show cursor
	if win.isHoveredInRewindArea {
		dl.AddRectFilled(imgui.Vec2{X: win.hoverX - traceCursorWidth/2, Y: rootPos.Y},
			imgui.Vec2{X: win.hoverX + traceCursorWidth/2, Y: rootPos.Y + traceSize.Y},
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
				thumbnailPos = imgui.Vec2{X: win.hoverX + traceCursorWidth*2, Y: rootPos.Y}
				if thumbnailPos.X+thumbnailSize.X > rootPos.X+traceSize.X {
					win.thmbFlippedCt++
					if win.thmbFlippedCt > int(win.img.plt.mode.RefreshRate/2) {
						win.thmbFlipped = false
						thumbnailPos.X = win.hoverX - traceCursorWidth*2 - thumbnailSize.X
					}
				} else {
					win.thmbFlippedCt = 0
				}
			} else {
				thumbnailPos = imgui.Vec2{X: win.hoverX - traceCursorWidth*2 - thumbnailSize.X, Y: rootPos.Y}
				if thumbnailPos.X < rootPos.X {
					win.thmbFlippedCt++
					if win.thmbFlippedCt > int(win.img.plt.mode.RefreshRate/2) {
						win.thmbFlipped = true
						thumbnailPos.X = win.hoverX + traceCursorWidth*2
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

	// draw frame guides
	var fn int
	if len(timeline.FrameNum) > 0 {
		fn = timeline.FrameNum[traceOffset] * tracePlotWidth
	}
	const guideFrameCount = 20
	imgui.PushFont(win.img.glsl.fonts.diagram)
	for i := 1 - (guideFrameCount * tracePlotWidth) - (fn % (guideFrameCount * tracePlotWidth)); i < traceFrameMax*tracePlotWidth; i += guideFrameCount * tracePlotWidth {
		top := imgui.Vec2{X: rootPos.X + float32(i), Y: rootPos.Y}
		bot := imgui.Vec2{X: rootPos.X + float32(i), Y: rootPos.Y + traceSize.Y}
		dl.AddRectFilled(top, bot, win.img.cols.timelineGuides)

		// label frame guides with frame numbers
		bot.X += 5
		bot.Y -= win.img.glsl.fonts.diagramSize / 2
		dl.AddText(bot, win.img.cols.timelineGuidesLabel, fmt.Sprintf("%d", (i-1)/tracePlotWidth))
	}
	imgui.PopFont()

	// draw main trace plot
	plotX := rootPos.X
	for i := range timeline.FrameNum[traceOffset:] {
		// adjust index by starting point
		i += traceOffset

		// SCANLINE TRACE
		plotY := yPos + traceMain

		// scale TotalScanlines value so that it covers the entire height of traceSize
		plotY -= float32(timeline.TotalScanlines[i]) * traceMain / specification.AbsoluteMaxScanlines

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
			imgui.Vec2{X: plotX + tracePlotWidth, Y: plotY + tracePlotHeight},
			win.img.cols.timelineScanlines)

		// WSYNC TRACE

		// plot WSYNC from the bottom
		plotY = yPos + traceMain
		plotY -= float32(timeline.Counts[i].WSYNC) * traceMain / specification.AbsoluteMaxClks

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
			imgui.Vec2{X: plotX + tracePlotWidth, Y: plotY + tracePlotHeight},
			win.img.cols.timelineWSYNC)

		// COPROCESSOR TRACE

		// plot coprocessor from the top
		if win.img.lz.Cart.HasCoProcBus {
			plotY = yPos
			plotY += float32(timeline.Counts[i].CoProc) * traceMain / specification.AbsoluteMaxClks

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
				imgui.Vec2{X: plotX + tracePlotWidth, Y: plotY + tracePlotHeight},
				win.img.cols.timelineCoProc)
		}

		plotX += tracePlotWidth
	}
	yPos += traceMain + traceGap

	// input trace
	// TODO: right player and panel input
	plotX = rootPos.X
	for i := range timeline.FrameNum[traceOffset:] {
		i += traceOffset
		if timeline.LeftPlayerInput[i] {
			dl.AddRectFilled(imgui.Vec2{X: plotX, Y: yPos},
				imgui.Vec2{X: plotX + tracePlotWidth, Y: yPos + traceInput},
				win.img.cols.timelineLeftPlayer)
		}
		plotX += tracePlotWidth
	}
	yPos += traceInput + traceGap

	// rewind range indicator
	dl.AddRectFilled(imgui.Vec2{X: rewindStartX, Y: yPos},
		imgui.Vec2{X: rewindEndX, Y: yPos + traceRewind},
		win.img.cols.timelineRewindRange)
	yPos += traceRewind + traceGap

	// jitter indicators
	for _, i := range scanlineJitter[1:] {
		dl.AddTriangleFilled(
			imgui.Vec2{
				X: rootPos.X + float32(i*tracePlotWidth) - traceFrame,
				Y: yPos + traceFrame*2,
			},
			imgui.Vec2{
				X: rootPos.X + float32(i*tracePlotWidth),
				Y: yPos,
			},
			imgui.Vec2{
				X: rootPos.X + float32(i*tracePlotWidth) + traceFrame,
				Y: yPos + traceFrame*2,
			},
			win.img.cols.timelineScanlines)
	}

	// comparison frame indicator
	if win.img.lz.Rewind.Comparison != nil {
		fr := win.img.lz.Rewind.Comparison.TV.GetCoords().Frame - rewindOffset

		if fr < 0 {
			// draw triangle indicating that the comparison frame is not
			// visible on the current timline
			dl.AddTriangleFilled(imgui.Vec2{X: rootPos.X - traceFrame, Y: yPos + traceFrame},
				imgui.Vec2{X: rootPos.X + traceFrame, Y: yPos + traceFrame*2},
				imgui.Vec2{X: rootPos.X + traceFrame, Y: yPos},
				win.img.cols.timelineCmpPointer)
		} else {
			dl.AddCircleFilled(imgui.Vec2{X: rootPos.X + float32(fr*tracePlotWidth), Y: yPos + traceFrame}, traceFrame, win.img.cols.timelineCmpPointer)
		}
	}

	// current frame indicator
	fr := win.img.lz.TV.Coords.Frame - rewindOffset
	dl.AddCircleFilled(imgui.Vec2{X: rootPos.X + float32(fr*tracePlotWidth), Y: yPos + traceFrame}, traceFrame, win.img.cols.timelineCurrentPointer)

	// end of trace area
	imgui.EndChild()

	// check for mouse hover over rewindable area
	mouse := imgui.MousePos()
	win.isHoveredInRewindArea = mouse.X >= rewindStartX && mouse.X <= rewindEndX &&
		mouse.Y >= rootPos.Y && mouse.Y <= rootPos.Y+traceSize.Y
	win.hoverX = mouse.X

	// rewind support
	rewindX := win.hoverX - rootPos.X
	rewindStartFrame := win.img.lz.Rewind.Timeline.AvailableStart
	rewindEndFrame := win.img.lz.Rewind.Timeline.AvailableEnd
	rewindHoverFrame := int(rewindX/tracePlotWidth) + rewindOffset

	if imgui.IsMouseDown(1) && imgui.IsItemHovered() {
		imgui.OpenPopup(timelinePopupID)

	} else if imgui.IsMouseDown(0) && (win.isHoveredInRewindArea || win.rewindingActive) {
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

		if imgui.IsItemHovered() && len(win.img.lz.Rewind.Timeline.FrameNum) > 0 {
			traceHoverIdx := int(rewindX/tracePlotWidth) + traceOffset
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

	if imgui.BeginPopup(timelinePopupID) {
		if imgui.Selectable(fmt.Sprintf("%c Save Timeline to CSV", fonts.Disk)) {
			win.saveToCSV()
		}
		imgui.EndPopup()
	}

	if win.isHoveredInRewindArea {
		// slow the rate at which we generate thumbnails
		if win.img.polling.throttleTimelineThumbnailer() {
			win.img.dbg.PushFunction(func() {
				// thumbnailer must be run in the same goroutine as the main emulation
				win.thmb.Create(win.img.dbg.Rewind.GetState(rewindHoverFrame))
			})
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
