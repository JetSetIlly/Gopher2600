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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/tracker"
)

const winTrackerID = "Audio Tracker"

type winTracker struct {
	playmodeWin
	debuggerWin

	img *SdlImgui

	contextMenu coords.TelevisionCoords

	// piano keys
	blackKeys       imgui.PackedColor
	whiteKeys       imgui.PackedColor
	whiteKeysGap    imgui.PackedColor
	pianoKeysHeight float32

	selection imguiSelection
}

func newWinTracker(img *SdlImgui) (window, error) {
	win := &winTracker{
		img:          img,
		blackKeys:    imgui.PackedColorFromVec4(imgui.Vec4{X: 0, Y: 0, Z: 0, W: 1.0}),
		whiteKeys:    imgui.PackedColorFromVec4(imgui.Vec4{X: 1.0, Y: 1.0, Z: 0.90, W: 1.0}),
		whiteKeysGap: imgui.PackedColorFromVec4(imgui.Vec4{X: 0.2, Y: 0.2, Z: 0.2, W: 1.0}),
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

	if imgui.BeginV(win.playmodeID(win.id()), &win.playmodeOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.playmodeWin.playmodeGeom.update()
	imgui.End()

	return true
}

const trackerContextMenuID = "trackerContextMenu"

func (win *winTracker) debuggerDraw() bool {
	if !win.debuggerIsOpen() {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 494, Y: 274}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 658, Y: 469}, imgui.ConditionFirstUseEver)
	win.img.setReasonableWindowConstraints()

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.debuggerWin.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winTracker) drawReplayButton() {
	s, e := win.selection.limits()
	if s == -1 || e == -1 || win.img.dbg.State() != govern.Paused {
		imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
		imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
		defer imgui.PopItemFlag()
		defer imgui.PopStyleVar()
	}

	if imgui.Button("Replay") {
		// unmute audio for the duration of the replay
		win.img.audio.Mute(false)

		win.img.dbg.Tracker.Replay(s, e, win.img.audio, func() {
			w, _ := time.ParseDuration("0.25s")
			time.Sleep(w)

			// which audio mute preference we're using depends on emulation mode
			var mutePrefs prefs.Bool
			if win.img.isPlaymode() {
				mutePrefs = win.img.prefs.audioMutePlaymode
			} else {
				mutePrefs = win.img.prefs.audioMuteDebugger
			}

			if mutePrefs.Get().(bool) {
				win.img.audio.Mute(true)
			}
		})
	}
}

func (win *winTracker) draw() {
	imgui.PushStyleColor(imgui.StyleColorTableHeaderBg, win.img.cols.AudioTrackerHeader)
	imgui.PushStyleColor(imgui.StyleColorTableBorderLight, win.img.cols.AudioTrackerBorder)
	imgui.PushStyleColor(imgui.StyleColorTableBorderStrong, win.img.cols.AudioTrackerBorder)
	imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.AudioTrackerRowHover)
	imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.AudioTrackerRowHover)
	defer imgui.PopStyleColorV(5)

	// get number of entries before anything else
	var numEntries int
	win.img.dbg.Tracker.BorrowTracker(func(history *tracker.History) {
		numEntries = len(history.Entries)
	})

	// draw toolbar if history is not empty
	if numEntries > 0 {
		// disable replay button as appropriate
		//
		// note that this is placed outside of the BorrowTracker() call below. this
		// is because the button will call the Replay() function which will try to
		// acquire the tracker lock, which has already been acquired by the borrow
		// process
		win.drawReplayButton()
	}

	win.img.dbg.Tracker.BorrowTracker(func(history *tracker.History) {
		// new child that contains the main scrollable table
		if imgui.BeginChildV("##trackerscroller", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.pianoKeysHeight}, false, 0) {
			if numEntries == 0 {
				imgui.Spacing()
				imgui.Text("Audio tracker history is empty")
			} else {
				// tracker table
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
					var altRowCol bool

					var clipper imgui.ListClipper
					clipper.Begin(numEntries)
					for clipper.Step() {
						for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
							entry := history.Entries[i]

							imgui.TableNextRow()
							altRowCol = !altRowCol

							if altRowCol {
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

							// add selectable for row. the text for the selectable
							// depends on which channel the tracker entry represents
							imgui.TableNextColumn()
							if entry.Channel == 1 || !entry.IsMusical() {
								imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.Transparent)
							}
							imgui.SelectableV(string(fonts.MusicNote), false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{X: 0, Y: 0})
							if entry.Channel == 1 || !entry.IsMusical() {
								imgui.PopStyleColor()
							}

							win.img.imguiTooltip(func() {
								imgui.Text(fmt.Sprintf("Frame: %d", entry.Coords.Frame))
								imgui.Text(fmt.Sprintf("Scanline: %d", entry.Coords.Scanline))
								imgui.Text(fmt.Sprintf("Clock: %d", entry.Coords.Clock))
							}, true)

							// context menu on right mouse button
							if imgui.IsItemHovered() {
								if imgui.IsMouseClicked(0) {
									win.selection.dragStart(i)
								}
								if imgui.IsMouseDragging(0, 0.0) {
									win.selection.drag(i)
								}
								if imgui.IsMouseDown(1) {
									imgui.OpenPopup(trackerContextMenuID)
									win.contextMenu = entry.Coords
								}
							}
							if entry.Coords == win.contextMenu {
								if imgui.BeginPopup(trackerContextMenuID) {
									if imgui.Selectable("Clear note history") {
										win.img.dbg.PushFunction(win.img.dbg.Tracker.Reset)
									}
									if imgui.Selectable(fmt.Sprintf("Rewind (to %s)", entry.Coords)) {
										win.img.dbg.GotoCoords(entry.Coords)
									}
									imgui.EndPopup()
								}
							}

							// if tracker entry is for channel one then skip the
							// first half-dozen columns and add whatever the 'note
							// icon' is
							if entry.Channel == 1 {
								imgui.TableNextColumn()
								imgui.TableNextColumn()
								imgui.TableNextColumn()
								imgui.TableNextColumn()
								imgui.TableNextColumn()
								imgui.TableNextColumn()
								if entry.IsMusical() {
									imgui.Text(string(fonts.MusicNote))
								}
							}

							imgui.TableNextColumn()
							imgui.Text(fmt.Sprintf("%04b", entry.Registers.Control&0x0f))
							imgui.TableNextColumn()
							imgui.Text(fmt.Sprintf("%s", entry.Distortion))
							imgui.TableNextColumn()
							imgui.Text(fmt.Sprintf("%05b", entry.Registers.Freq&0x1f))
							imgui.TableNextColumn()

							// convert musical note into something worth showing
							musicalNote := string(entry.MusicalNote)
							switch entry.MusicalNote {
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

							// volum column
							var volumeArrow rune

							switch entry.Volume {
							case tracker.VolumeRising:
								volumeArrow = fonts.VolumeRising
							case tracker.VolumeFalling:
								volumeArrow = fonts.VolumeFalling
							}

							imgui.Text(fmt.Sprintf("%02d %c", entry.Registers.Volume&0x4b, volumeArrow))
						}
					}

					if win.img.dbg.State() == govern.Running {
						imgui.SetScrollHereY(1.0)
					}

					imgui.EndTable()
				}
			}
		}
		imgui.EndChild()

		// don't allow grabbing or movement of window when piano keys are clicked
		if imgui.BeginChildV("##pianokeys", imgui.Vec2{}, false, imgui.WindowFlagsNoMove) {
			win.pianoKeysHeight = win.drawPianoKeys(history)
		}
		imgui.EndChild()
	})
}
