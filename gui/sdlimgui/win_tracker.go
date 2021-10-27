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
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/tracker"
)

const winTrackerID = "Audio Tracker"

type winTracker struct {
	img  *SdlImgui
	open bool

	footerHeight float32
	contextMenu  coords.TelevisionCoords
}

func newWinTracker(img *SdlImgui) (window, error) {
	win := &winTracker{
		img: img,
	}
	return win, nil
}

func (win *winTracker) init() {
	// nominal value to stop scrollbar appearing for a frame (it takes a
	// frame before we set the correct footerHeight value
	win.footerHeight = imgui.FrameHeight() + imgui.CurrentStyle().FramePadding().Y
}

func (win *winTracker) id() string {
	return winTrackerID
}

func (win *winTracker) isOpen() bool {
	return win.open
}

func (win *winTracker) setOpen(open bool) {
	win.open = open
}

const trackerContextMenuID = "trackerContextMenu"

func (win *winTracker) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{574, 97}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{496, 614}, imgui.ConditionFirstUseEver)

	imgui.BeginV(win.id(), &win.open, 0)
	defer imgui.End()

	imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.DisasmHover)
	imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.DisasmHover)
	defer imgui.PopStyleColorV(2)

	imgui.PushStyleColor(imgui.StyleColorTableHeaderBg, win.img.cols.AudioTrackerHeader)
	defer imgui.PopStyleColor()

	numEntries := len(win.img.lz.Tracker.Entries)
	if numEntries == 0 {
		imgui.Text("No audio output/changes yet")
	} else {
		tableFlags := imgui.TableFlagsNone
		tableFlags |= imgui.TableFlagsSizingFixedFit
		tableFlags |= imgui.TableFlagsBordersV
		tableFlags |= imgui.TableFlagsBordersOuter

		// I can't get the header of the table to freeze in the scroller so I'm
		// fudging the effect by having a separate table just for the header.
		if !imgui.BeginTableV("trackerHeader", 10, tableFlags, imgui.Vec2{}, 0) {
			return
		}
		imgui.TableSetupColumnV("", imgui.TableColumnFlagsNone, 0, 0)
		imgui.TableSetupColumnV("0", imgui.TableColumnFlagsNone, 15, 1)
		imgui.TableSetupColumnV("Control", imgui.TableColumnFlagsNone, 100, 2)
		imgui.TableSetupColumnV("Freq", imgui.TableColumnFlagsNone, 40, 3)
		imgui.TableSetupColumnV("Vol", imgui.TableColumnFlagsNone, 30, 4)
		imgui.TableSetupColumnV("", imgui.TableColumnFlagsNone, 0, 5)
		imgui.TableSetupColumnV("1", imgui.TableColumnFlagsNone, 15, 6)
		imgui.TableSetupColumnV("Control", imgui.TableColumnFlagsNone, 100, 7)
		imgui.TableSetupColumnV("Freq", imgui.TableColumnFlagsNone, 40, 8)
		imgui.TableSetupColumnV("Vol", imgui.TableColumnFlagsNone, 30, 9)
		imgui.TableHeadersRow()
		imgui.EndTable()

		// new child that contains the main scrollable table
		imgui.BeginChildV("##trackerscroller", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.footerHeight}, false, 0)

		if !imgui.BeginTableV("tracker", 10, tableFlags, imgui.Vec2{}, 0) {
			return
		}

		imgui.TableSetupColumnV("", imgui.TableColumnFlagsNone, 0, 0)
		imgui.TableSetupColumnV("0", imgui.TableColumnFlagsNone, 15, 1)
		imgui.TableSetupColumnV("Control", imgui.TableColumnFlagsNone, 100, 2)
		imgui.TableSetupColumnV("Freq", imgui.TableColumnFlagsNone, 40, 3)
		imgui.TableSetupColumnV("Vol", imgui.TableColumnFlagsNone, 30, 4)
		imgui.TableSetupColumnV("", imgui.TableColumnFlagsNone, 0, 5)
		imgui.TableSetupColumnV("1", imgui.TableColumnFlagsNone, 15, 6)
		imgui.TableSetupColumnV("Control", imgui.TableColumnFlagsNone, 100, 7)
		imgui.TableSetupColumnV("Freq", imgui.TableColumnFlagsNone, 40, 8)
		imgui.TableSetupColumnV("Vol", imgui.TableColumnFlagsNone, 30, 9)

		// altenate row colors at change of frame number
		var lastEntry tracker.Entry
		var lastEntryChan0 tracker.Entry
		var lastEntryChan1 tracker.Entry
		var altRowCol bool

		var clipper imgui.ListClipper
		clipper.Begin(numEntries)
		for clipper.Step() {
			for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
				entry := win.img.lz.Tracker.Entries[i]

				imgui.TableNextRow()

				// flip row color
				if entry.Coords.Frame != lastEntry.Coords.Frame {
					altRowCol = !altRowCol
				}

				if altRowCol {
					imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, win.img.cols.AudioTrackerRowAlt)
					imgui.TableSetBgColor(imgui.TableBgTargetRowBg1, win.img.cols.AudioTrackerRowAlt)
				} else {
					imgui.TableSetBgColor(imgui.TableBgTargetRowBg0, win.img.cols.AudioTrackerRow)
					imgui.TableSetBgColor(imgui.TableBgTargetRowBg1, win.img.cols.AudioTrackerRow)
				}

				imgui.TableNextColumn()
				imgui.SelectableV("", false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
				if imgui.IsItemHovered() {
					imgui.BeginTooltip()
					imgui.Text(fmt.Sprintf("Frame: %d", entry.Coords.Frame))
					imgui.Text(fmt.Sprintf("Scanline: %d", entry.Coords.Scanline))
					imgui.Text(fmt.Sprintf("Clock: %d", entry.Coords.Clock))
					imgui.EndTooltip()
				}
				// context menu on right mouse button
				if imgui.IsItemHovered() && imgui.IsMouseDown(1) {
					imgui.OpenPopup(trackerContextMenuID)
					win.contextMenu = entry.Coords
				}
				if entry.Coords == win.contextMenu {
					if imgui.BeginPopup(trackerContextMenuID) {
						if imgui.Selectable("Rewind to") {
							win.img.dbg.PushGoto(entry.Coords)
						}
						imgui.EndPopup()
					}
				}

				if entry.Channel == 1 {
					imgui.TableNextColumn()
					imgui.TableNextColumn()
					imgui.TableNextColumn()
					imgui.TableNextColumn()
					imgui.TableNextColumn()
				}

				imgui.TableNextColumn()
				if entry.MusicalNote != tracker.NoMusicalNote {
					imgui.Text(fmt.Sprintf("%c", fonts.MusicNote))
				}
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("(%01x) %s", entry.Registers.Control&0x0f, entry.Distortion))
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("%05b", entry.Registers.Freq))
				imgui.TableNextColumn()

				// volum column
				var volumeArrow rune

				// compare with previous entry for the channel
				if entry.Channel == 0 {
					if entry.Registers.Volume > lastEntryChan0.Registers.Volume {
						volumeArrow = fonts.VolumeUp
					} else if entry.Registers.Volume < lastEntryChan0.Registers.Volume {
						volumeArrow = fonts.VolumeDown
					}
					lastEntryChan0 = entry
				} else {
					if entry.Registers.Volume > lastEntryChan1.Registers.Volume {
						volumeArrow = fonts.VolumeUp
					} else if entry.Registers.Volume < lastEntryChan1.Registers.Volume {
						volumeArrow = fonts.VolumeDown
					}
					lastEntryChan1 = entry
				}

				imgui.Text(fmt.Sprintf("%02d %c", entry.Registers.Volume&0x4b, volumeArrow))

				// record last entry for comparison purposes next iteration
				lastEntry = entry

			}
		}

		imgui.EndTable()

		if win.img.emulation.State() == emulation.Running {
			imgui.SetScrollHereY(1.0)
		}

		imgui.EndChild()

		win.footerHeight = imguiMeasureHeight(func() {
			imgui.Spacing()

			imgui.AlignTextToFramePadding()
			imgui.Text(fmt.Sprintf("Last change: %s", lastEntry.Coords))

			imgui.SameLineV(0, 15)
			if imgui.Button("Rewind to") {
				win.img.dbg.PushGoto(lastEntry.Coords)
			}
		})
	}
}
