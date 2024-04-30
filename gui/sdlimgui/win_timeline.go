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
	"image"
	"os"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources/unique"
	"github.com/jetsetilly/gopher2600/thumbnailer"
	"golang.org/x/image/draw"
)

const winTimelineID = "Timeline"

type winTimeline struct {
	debuggerWin

	img *SdlImgui

	// thumbnailer will be using emulation states created in the main emulation
	// goroutine so we must thumbnail those states in the same goroutine.
	thmb        *thumbnailer.Image
	thmbTexture texture

	// the backing image for the texture
	thmbImage      *image.RGBA
	thmbDimensions image.Point

	// whether the thumbnail is being shown on the left of the timeline rather
	// than the right
	thumbLeft bool

	// which frame was the last thumbnail generated for. used to prevent another
	// thumbnail being generated for the same frame
	thmbFrame int

	// throttle the number of thumbnails that are being generated at once. if
	// channel is full then the thumbnail is being generated. the channel is
	// drained once the thumbnail creation has completed
	thmbRunning chan bool

	// mouse hover information
	isHovered  bool
	hoverX     float32
	hoverIdx   int
	hoverFrame int

	// mouse is being used to scrub the timeline area. see isScrubbingValid()
	// for a function that is a more likely to provide a useful value
	scrubbing bool

	// the following two fields are used to help understand what to do when the
	// timeline window is resized
	//
	// the trace offset from the previous frame
	//
	// pushing flag is true if the current frame indicator is near the right
	// most limit of the trace area. it's used to
	pushing         bool
	prevTraceOffset int

	// height of toolbar
	toolbarHeight float32
}

func newWinTimeline(img *SdlImgui) (window, error) {
	win := &winTimeline{
		img:         img,
		thmbRunning: make(chan bool, 1),
	}

	var err error

	win.thmb, err = thumbnailer.NewImage(win.img.dbg.VCS().Env.Prefs)
	if err != nil {
		return nil, fmt.Errorf("debugger: %w", err)
	}

	win.thmbTexture = img.rnd.addTexture(textureColor, true, true)
	win.thmbImage = image.NewRGBA(image.Rect(0, 0, specification.ClksVisible, specification.AbsoluteMaxScanlines))
	win.thmbDimensions = win.thmbImage.Bounds().Size()

	return win, nil
}

func (win *winTimeline) isScrubbingValid() bool {
	return win.scrubbing && win.hoverIdx >= 0 && win.hoverIdx < len(win.img.cache.Rewind.Timeline.FrameNum)
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
	case newImage := <-win.thmb.Render:
		if newImage != nil {
			// clear image
			for i := 0; i < len(win.thmbImage.Pix); i += 4 {
				s := win.thmbImage.Pix[i : i+4 : i+4]
				s[0] = 10
				s[1] = 10
				s[2] = 10
				s[3] = 255
			}

			// copy new image so that it is centred in the thumbnail image
			sz := newImage.Bounds().Size()
			y := ((win.thmbDimensions.Y - sz.Y) / 2)
			draw.Copy(win.thmbImage, image.Point{X: 0, Y: y},
				newImage, newImage.Bounds(), draw.Over, nil)
			win.thmbTexture.render(win.thmbImage)
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
				imgui.Spacing()
				imgui.Separator()
				imgui.Spacing()

				if imgui.Selectable(fmt.Sprintf("Set Comparison to Frame %d", win.hoverFrame)) {
					win.img.dbg.PushFunction(func() {
						win.img.term.pushCommand(fmt.Sprintf("COMPARISON %d", win.hoverFrame))
					})
				}

				var label string
				var command string
				if win.img.cache.Rewind.Comparison.Locked {
					label = "Unlock Comparison Frame"
					command = "COMPARISON UNLOCK"
				} else {
					label = fmt.Sprintf("Lock Comparison Frame (at frame %d)",
						win.img.cache.Rewind.Comparison.State.TV.GetCoords().Frame)
					command = "COMPARISON LOCK"
				}
				if imgui.Selectable(label) {
					win.img.term.pushCommand(command)
				}
			}
			imgui.EndPopup()
		}
	}

	win.debuggerGeom.update()
	imgui.End()

	if (win.isHovered || win.isScrubbingValid()) && win.img.prefs.showTimelineThumbnail.Get().(bool) {
		if win.thmbFrame != win.hoverFrame {
			select {
			case win.thmbRunning <- true:
				win.thmbFrame = win.hoverFrame
				hoverFrame := win.hoverFrame
				win.img.dbg.PushFunction(func() {
					// thumbnailer must be run in the same goroutine as the main emulation
					win.thmb.Create(win.img.dbg.Rewind.GetState(hoverFrame))
					<-win.thmbRunning
				})
			default:
				// if a thumbnail is currently being generated then we need to
				// carry on with the GUI thread without delay
			}
		}
	}

	return true
}

func (win *winTimeline) drawToolbar() {
	timeline := win.img.cache.Rewind.Timeline
	if timeline.AvailableStart == timeline.AvailableEnd && timeline.AvailableStart == 0 {
		imgui.Text("No rewind history")
	} else {
		imgui.Text(fmt.Sprintf("Rewind between %d and %d", timeline.AvailableStart, timeline.AvailableEnd))
		imgui.SameLineV(0, 15)
		imguiColorLabelSimple(fmt.Sprintf("Comparing frame %d", win.img.cache.Rewind.Comparison.State.TV.GetCoords().Frame), win.img.cols.TimelineComparison)

		if win.isHovered || win.isScrubbingValid() {
			imgui.SameLineV(0, 15)
			imguiColorLabelSimple(fmt.Sprintf("%d Scanlines", win.img.cache.Rewind.Timeline.TotalScanlines[win.hoverIdx]), win.img.cols.TimelineScanlines)

			imgui.SameLineV(0, 15)
			imguiColorLabelSimple(fmt.Sprintf("%.02f%% WSYNC", win.img.cache.Rewind.Timeline.Ratios[win.hoverIdx].WSYNC*100), win.img.cols.TimelineWSYNC)

			if win.img.cache.VCS.Mem.Cart.GetCoProcBus() != nil {
				imgui.SameLineV(0, 15)
				imguiColorLabelSimple(fmt.Sprintf("%.02f%% Coproc", win.img.cache.Rewind.Timeline.Ratios[win.hoverIdx].CoProc*100), win.img.cols.TimelineCoProc)
			}
		}
	}
}

func (win *winTimeline) drawTrace() {
	timeline := win.img.cache.Rewind.Timeline
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
	iconRadius := win.img.fonts.guiSize / 2

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
	var traceOffset int
	if len(timeline.FrameNum)*plotWidth >= int(availableWidth) {
		traceOffset = len(timeline.FrameNum) - availableWidthInFrames
	}
	if win.pushing || traceOffset > win.prevTraceOffset {
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
	imgui.PushFont(win.img.fonts.diagram)

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
			bot.Y -= win.img.fonts.diagramSize / 2
			dl.AddText(bot, win.img.cols.timelineGuidesLabel, fmt.Sprintf("%d", fn))
		}
		guideX += plotWidth
	}
	imgui.PopFont()

	// show hover/scrubbing cursor
	if win.isHovered || win.isScrubbingValid() {
		dl.AddRectFilled(imgui.Vec2{X: win.hoverX - cursorWidth/2, Y: rootPos.Y},
			imgui.Vec2{X: win.hoverX + cursorWidth/2, Y: rootPos.Y + traceSize.Y},
			win.img.cols.timelineHoverCursor)

		if win.img.prefs.showTimelineThumbnail.Get().(bool) {
			sz := imgui.Vec2{float32(win.thmbDimensions.X) * 2, float32(win.thmbDimensions.Y)}
			sz = sz.Times(traceSize.Y / specification.AbsoluteMaxScanlines)

			// show thumbnail on either the left or right of the timeline window
			var pos imgui.Vec2
			if win.thumbLeft {
				pos = imgui.Vec2{X: rootPos.X + iconRadius*2, Y: rootPos.Y}
				if win.hoverX <= pos.X+sz.X {
					win.thumbLeft = false
					pos.X = rootPos.X + availableWidth - sz.X - iconRadius*2
				}
			} else {
				pos = imgui.Vec2{X: rootPos.X + availableWidth - sz.X - iconRadius*2, Y: rootPos.Y}
				if win.hoverX >= pos.X {
					win.thumbLeft = true
					pos.X = rootPos.X + iconRadius*2
				}
			}
			imgui.SetCursorScreenPos(pos)

			imgui.ImageV(imgui.TextureID(win.thmbTexture.getID()), sz,
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
		if win.img.cache.VCS.Mem.Cart.GetCoProcBus() != nil {
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
		pos := imgui.Vec2{X: rootPos.X + float32(i*plotWidth), Y: yPos}
		dl.AddText(pos, win.img.cols.timelineScanlines, string(fonts.TimelineJitter))
	}

	// comparison frame indicator
	if win.img.cache.Rewind.Comparison.State != nil && len(timeline.FrameNum) > 0 {
		fr := win.img.cache.Rewind.Comparison.State.TV.GetCoords().Frame - rewindOffset

		if fr < 0 {
			// indicate that the comparison frame is not visible
			pos := imgui.Vec2{X: rootPos.X - iconRadius, Y: yPos}
			dl.AddText(pos, win.img.cols.timelineComparison, string(fonts.TimelineOffScreen))
		} else {
			pos := imgui.Vec2{X: rootPos.X + float32(fr*plotWidth) - iconRadius, Y: yPos}
			if win.img.cache.Rewind.Comparison.Locked {
				dl.AddText(pos, win.img.cols.timelineComparison, string(fonts.TimelineComparisonLock))
			} else {
				dl.AddText(pos, win.img.cols.timelineComparison, string(fonts.TimelineComparison))
			}
		}
	}

	// current frame indicator
	currentFrame := win.img.cache.TV.GetCoords().Frame - rewindOffset
	if currentFrame < 0 {
		// indicate that the current frame indicator is not visible
		pos := imgui.Vec2{X: rootPos.X - iconRadius, Y: yPos}
		dl.AddText(pos, win.img.cols.timelineRewindRange, string(fonts.TimelineOffScreen))
	} else {
		dl.AddText(imgui.Vec2{X: rootPos.X + float32(currentFrame*plotWidth) - iconRadius, Y: yPos},
			win.img.cols.timelineRewindRange, string(fonts.TV))
	}
	win.pushing = float32(currentFrame*plotWidth) >= availableWidth*0.99

	// end of trace area
	imgui.EndChild()

	// no mouse handling for timeline window if popup window is open
	if imgui.IsPopupOpen(timelinePopupID) {
		return
	}

	// detect mouse is in scrubbing areas
	win.scrubbing = (win.scrubbing && imgui.IsMouseDown(0)) ||
		(imgui.IsItemHovered() && imgui.IsMouseClicked(0))

	// check for mouse hover over rewindable area
	mouse := imgui.MousePos()
	win.isHovered = (imgui.IsItemHovered() &&
		mouse.X >= rewindStartX && mouse.X <= rewindEndX &&
		mouse.Y >= rootPos.Y && mouse.Y <= rootPos.Y+traceSize.Y)
	win.hoverX = mouse.X

	// rewind support
	rewindX := win.hoverX - rootPos.X
	rewindStartFrame := win.img.cache.Rewind.Timeline.AvailableStart
	rewindEndFrame := win.img.cache.Rewind.Timeline.AvailableEnd

	// index and frame number for hover position
	win.hoverIdx = int(rewindX/plotWidth) + traceOffset
	win.hoverFrame = int(rewindX/plotWidth) + rewindOffset

	// mouse handling
	if imgui.IsMouseDown(1) && imgui.IsItemHovered() {
		imgui.OpenPopup(timelinePopupID)

	} else if win.isScrubbingValid() {
		coords := win.img.cache.TV.GetCoords()

		// making sure we only call PushRewind() when we need to. also,
		// allowing mouse to travel beyond the rewind boundaries (and without
		// calling PushRewind() too often)
		if win.hoverFrame >= rewindEndFrame {
			if coords.Frame < rewindEndFrame {
				win.img.dbg.RewindToFrame(win.hoverFrame, true)
			}
		} else if win.hoverFrame <= rewindStartFrame {
			if coords.Frame > rewindStartFrame {
				win.img.dbg.RewindToFrame(win.hoverFrame, false)
			}
		} else if win.hoverFrame != coords.Frame {
			win.img.dbg.RewindToFrame(win.hoverFrame, win.hoverFrame == rewindEndFrame)
		}
	}
}

func (win *winTimeline) saveToCSV() {
	// open unique file
	fn := unique.Filename("timeline", win.img.cache.VCS.Mem.Cart.ShortName)
	fn = fmt.Sprintf("%s.csv", fn)
	f, err := os.Create(fn)
	if err != nil {
		logger.Logf(logger.Allow, "sdlimgui", "could not save timeline CSV: %v", err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Logf(logger.Allow, "sdlimgui", "error saving timeline CSV: %v", err)
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

	timeline := win.img.cache.Rewind.Timeline
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
