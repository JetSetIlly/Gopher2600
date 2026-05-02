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
	"time"

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/tracker"
	"github.com/jetsetilly/imgui-go/v5"
)

const winTrackerID = "Audio Tracker"

type winTracker struct {
	playmodeWin
	debuggerWin

	img *SdlImgui

	// piano keys
	blackKeys       imgui.PackedColor
	whiteKeys       imgui.PackedColor
	whiteKeysGap    imgui.PackedColor
	pianoKeysHeight float32

	// the amount of the tracker listing that has been selected
	selection imguiSelection

	// the entry that should be included in the popup context menu
	contextEntry *tracker.Entry

	// trigger to replay the selected area
	//
	// we use this to communicate to the window from within the BorrowTracker() function. the
	// Replay() function also locks the tracker listing so calling Replay from within
	// BorrowTracker() will cause a deadlock
	//
	// there are other ways to resolve this but this is the way I have chosen
	replay chan bool
}

func newWinTracker(img *SdlImgui) (window, error) {
	win := &winTracker{
		img:          img,
		blackKeys:    imgui.PackedColorFromVec4(imgui.Vec4{X: 0, Y: 0, Z: 0, W: 1.0}),
		whiteKeys:    imgui.PackedColorFromVec4(imgui.Vec4{X: 1.0, Y: 1.0, Z: 0.90, W: 1.0}),
		whiteKeysGap: imgui.PackedColorFromVec4(imgui.Vec4{X: 0.2, Y: 0.2, Z: 0.2, W: 1.0}),
		replay:       make(chan bool, 1),
	}
	win.selection.clear()
	return win, nil
}

func (win *winTracker) init() {
	// nominal value to stop scrollbar appearing for a frame (it takes a
	// frame before we set the correct footerHeight value
	win.pianoKeysHeight = imgui.FrameHeight() + imgui.CurrentStyle().FramePadding().Y
}

func (win *winTracker) id() string {
	return winTrackerID
}

func (win *winTracker) playmodeDraw() bool {
	if !win.playmodeIsOpen() {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 494, Y: 274}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 658, Y: 469}, imgui.ConditionFirstUseEver)
	win.img.setReasonableWindowConstraints()

	var flgs imgui.WindowFlags
	flgs = imgui.WindowFlagsNone

	if imgui.BeginV(win.playmodeID(win.id()), &win.playmodeOpen, flgs) {
		win.draw()
	}

	win.playmodeWin.playmodeGeom.update()
	imgui.End()

	return true
}

const trackerContextID = "trackerContextMenu"

func (win *winTracker) debuggerDraw() bool {
	if !win.debuggerIsOpen() {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 494, Y: 274}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 658, Y: 469}, imgui.ConditionFirstUseEver)
	win.img.setReasonableWindowConstraints()

	var flgs imgui.WindowFlags
	flgs = imgui.WindowFlagsNone

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, flgs) {
		win.draw()
	}

	win.debuggerWin.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winTracker) draw() {
	imgui.PushStyleColor(imgui.StyleColorTableHeaderBg, win.img.cols.AudioTrackerHeader)
	imgui.PushStyleColor(imgui.StyleColorTableBorderLight, win.img.cols.AudioTrackerBorder)
	imgui.PushStyleColor(imgui.StyleColorTableBorderStrong, win.img.cols.AudioTrackerBorder)
	imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.AudioTrackerRowHover)
	imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.AudioTrackerRowHover)
	defer imgui.PopStyleColorV(5)

	win.img.dbg.Tracker.BorrowTracker(win.drawListing)

	select {
	case <-win.replay:
		win.img.audio.Mute(false)

		s, e := win.selection.limits()
		win.img.dbg.Tracker.Replay(s, e, win.img.audio, func() {
			w, _ := time.ParseDuration("0.25s")
			time.Sleep(w)

			var m prefs.Bool
			if win.img.isPlaymode() {
				m = win.img.prefs.audioMutePlaymode
			} else {
				m = win.img.prefs.audioMuteDebugger
			}
			if m.Get().(bool) {
				win.img.audio.Mute(true)
			}
		})
	default:
	}
}

func (win *winTracker) drawListing(history *tracker.Listing) {
	numEntries := len(history.Entries)

	// draw toolbar if history is not empty
	if numEntries > 0 {
		s, e := win.selection.limits()
		drawDisabled(s == -1 || e == -1 || win.img.dbg.State() != govern.Paused, func() {
			if imgui.Button("Replay") {
				select {
				case win.replay <- true:
				default:
				}
			}
		})
	}

	// new child that contains the main scrollable table
	if imgui.BeginChildV("##trackerscroller", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.pianoKeysHeight}, false, 0) {
		if numEntries == 0 {
			imgui.Spacing()
			imgui.Text("Audio tracker history is empty")
		} else {
			const numColumns = 12
			flgs := imgui.TableFlagsScrollY
			flgs |= imgui.TableFlagsSizingStretchProp
			flgs |= imgui.TableFlagsNoHostExtendX

			if imgui.BeginTableV("tracker", numColumns, flgs, imgui.Vec2{}, 0) {
				imgui.TableSetupColumnV("", imgui.TableColumnFlagsNone, 15, 0)
				imgui.TableSetupColumnV("AUDC0", imgui.TableColumnFlagsNone, 40, 1)
				imgui.TableSetupColumnV("Distortion", imgui.TableColumnFlagsNone, 80, 2)
				imgui.TableSetupColumnV("AUDF0", imgui.TableColumnFlagsNone, 40, 3)
				imgui.TableSetupColumnV("Note", imgui.TableColumnFlagsNone, 30, 4)
				imgui.TableSetupColumnV("AUDV0", imgui.TableColumnFlagsNone, 40, 5)
				imgui.TableSetupColumnV("", imgui.TableColumnFlagsNone, 15, 6)
				imgui.TableSetupColumnV("AUDC1", imgui.TableColumnFlagsNone, 40, 7)
				imgui.TableSetupColumnV("Distortion", imgui.TableColumnFlagsNone, 80, 8)
				imgui.TableSetupColumnV("AUDF1", imgui.TableColumnFlagsNone, 40, 9)
				imgui.TableSetupColumnV("Note", imgui.TableColumnFlagsNone, 30, 10)
				imgui.TableSetupColumnV("AUDV1", imgui.TableColumnFlagsNone, 40, 11)

				imgui.TableSetupScrollFreeze(0, 1)
				imgui.TableHeadersRow()

				// altenate row colors at change of frame number
				var lastFrame int
				var alt bool

				imgui.ListClipperAll(numEntries, func(i int) {
					e := history.Entries[i]

					imgui.TableNextRow()

					if e.Coords.Frame != lastFrame {
						lastFrame = e.Coords.Frame
						alt = !alt
					}

					if alt {
						imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, win.img.cols.AudioTrackerRowAlt)
						imgui.TableSetBgColor(imgui.TableBgTargetRowBg1, win.img.cols.AudioTrackerRowAlt)
						if win.selection.inRange(i) {
							imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, win.img.cols.AudioTrackerRowSelectedAlt)
							imgui.TableSetBgColor(imgui.TableBgTargetRowBg1, win.img.cols.AudioTrackerRowSelectedAlt)
						}
					} else {
						imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, win.img.cols.AudioTrackerRow)
						imgui.TableSetBgColor(imgui.TableBgTargetRowBg1, win.img.cols.AudioTrackerRow)
						if win.selection.inRange(i) {
							imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, win.img.cols.AudioTrackerRowSelected)
							imgui.TableSetBgColor(imgui.TableBgTargetRowBg1, win.img.cols.AudioTrackerRowSelected)
						}
					}

					// add selectable for row. the text for the selectable depends on which channel
					// the tracker entry represents
					imgui.TableNextColumn()
					if e.Channel == 1 || !e.IsMusical() {
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.Transparent)
					}
					imgui.SelectableV(string(fonts.MusicNote), false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{X: 0, Y: 0})
					if e.Channel == 1 || !e.IsMusical() {
						imgui.PopStyleColor()
					}

					win.img.imguiTooltip(func() {
						imgui.Text(fmt.Sprintf("Frame: %d", e.Coords.Frame))
						imgui.Text(fmt.Sprintf("Scanline: %d", e.Coords.Scanline))
						imgui.Text(fmt.Sprintf("Clock: %d", e.Coords.Clock))
					}, true)

					// context menu on right mouse button
					if imgui.IsItemHovered() {
						if imgui.IsMouseClicked(0) {
							// if ctrl key is pressed then extend selection
							if imgui.CurrentIO().KeyShiftPressed() {
								win.selection.drag(i)
							} else {
								win.selection.dragStart(i)
							}
						}

						if imgui.IsMouseDragging(0, 0.0) {
							win.selection.drag(i)
						}

						if imgui.IsMouseClicked(1) {
							imgui.OpenPopup(trackerContextID)
							win.contextEntry = &e
						}
					}

					// if tracker entry is for channel one then skip the first half-dozen columns
					// and add whatever the 'note icon' is
					if e.Channel == 1 {
						imgui.TableNextColumn()
						imgui.TableNextColumn()
						imgui.TableNextColumn()
						imgui.TableNextColumn()
						imgui.TableNextColumn()
						imgui.TableNextColumn()
						if e.IsMusical() {
							imgui.Text(string(fonts.MusicNote))
						}
					}

					imgui.TableNextColumn()
					imgui.Text(fmt.Sprintf("%04b", e.Registers.Control))
					imgui.TableNextColumn()
					imgui.Text(fmt.Sprintf("%s", e.Distortion))
					imgui.TableNextColumn()
					imgui.Text(fmt.Sprintf("%05b", e.Registers.Freq))
					imgui.TableNextColumn()

					// convert musical note into something worth showing
					musicalNote := string(e.MusicalNote)
					switch e.MusicalNote {
					case tracker.Noise:
						musicalNote = ""
					case tracker.Low:
						musicalNote = ""
					case tracker.Silence:
						musicalNote = ""
					default:
						imgui.Text(musicalNote)
					}

					imgui.TableNextColumn()

					// volume column
					var volumeArrow rune
					switch e.Volume {
					case tracker.VolumeRising:
						volumeArrow = fonts.VolumeRising
					case tracker.VolumeFalling:
						volumeArrow = fonts.VolumeFalling
					}
					imgui.Text(fmt.Sprintf("%02d %c", e.Registers.Volume, volumeArrow))
				})

				if win.img.dbg.State() == govern.Running {
					imgui.SetScrollHereY(1.0)
				}

				// draw popup menu inside the table but outside the ListClipper loop
				if imgui.BeginPopup(trackerContextID) {
					if imgui.Selectable("Clear note history") {
						win.img.dbg.PushFunction(win.img.dbg.Tracker.Reset)
					}
					drawDisabled(history.Stable && history.Balanced, func() {
						if imgui.Selectable("Export to .tia") {
							history.Export(tracker.ExportTIA, win.img.cache.VCS.Mem.Cart.ShortName)
						}
					})
					if win.contextEntry != nil {
						if !win.img.isPlaymode() && win.img.dbg.State() == govern.Paused {
							imgui.Spacing()
							imgui.Separator()
							imgui.Spacing()
							if imgui.Selectable(fmt.Sprintf("Rewind (to %s)", win.contextEntry.Coords)) {
								win.img.dbg.GotoCoords(win.contextEntry.Coords)
							}
						}
					}
					imgui.EndPopup()
				}

				imgui.EndTable()
			}
		}
	}

	imgui.EndChild()

	if imgui.BeginChildV("##pianokeys", imgui.Vec2{}, false, imgui.ChildFlagsNone) {
		win.pianoKeysHeight = win.drawPianoKeys(history)
	}
	imgui.EndChild()
}
