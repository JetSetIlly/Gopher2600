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
	"strconv"

	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/imgui-go/v5"
)

// draw grid of bytes with before and after functions in addition to commit function.
func drawByteGrid(id string, data []uint8, origin uint32,
	before func(idx int), after func(idx int), commit func(idx int, value uint8)) {

	// the origin and memtop as a string
	originString := fmt.Sprintf("%08x", origin)
	memtopString := fmt.Sprintf("%08x", origin+uint32(len(data)-1))

	// find first non-matching digit of origin and memtop strings
	columnCrop := 0
	for i := 0; i < len(originString); i++ {
		if originString[i] != memtopString[i] {
			columnCrop = i
			break // for loop
		}
	}

	// the width of the row heading column
	rowHeadingWidth := len(originString) - columnCrop

	spacing := imgui.Vec2{X: 0.5, Y: 0.5}
	imgui.PushStyleVarVec2(imgui.StyleVarCellPadding, spacing)
	defer imgui.PopStyleVar()

	const numColumns = 16

	flgs := imgui.TableFlagsSizingFixedFit

	if imgui.BeginTableV(id, numColumns+1, flgs, imgui.Vec2{}, 0.0) {
		// in some situations we will return early from the drawByteGrid()
		// function so we want to make sure that EndTable() is called
		defer imgui.EndTable()

		imgui.TableSetupScrollFreeze(0, 1)

		// set up columns
		width := imguiTextWidth(rowHeadingWidth)
		imgui.TableSetupColumnV(fmt.Sprintf("%p_column0", data), imgui.TableColumnFlagsNone, width, 0)
		width = imguiTextWidth(2)
		for i := 1; i < numColumns+1; i++ {
			imgui.TableSetupColumnV(fmt.Sprintf("%p_column%d", data, i), imgui.TableColumnFlagsNone, width, 0)
		}

		// header row
		imgui.TableNextRow()

		// skip first column of the header row
		imgui.TableNextColumn()

		// try to center header with the text in the column
		leftPad := imgui.CurrentStyle().FramePadding().X

		// draw headers for each column
		for i := range numColumns {
			imgui.TableNextColumn()
			pos := imgui.CursorPos()
			pos.X += leftPad
			imgui.SetCursorPos(pos)
			imgui.Text(fmt.Sprintf("-%x", i))
		}

		// simple way of creating a gap to the main body of the table
		imgui.TableNextRow()
		imgui.TableNextRow()

		// the number of leading columns is the number of empty columns on the
		// first row
		//
		// we need to account for these leading columns when:
		// a) calculating the clipper length value
		// b) setting the idx and address values at the start of every row
		leadingColumns := int(origin % numColumns)

		// first row requires special handling in order to account for blank
		// columns on the first row
		firstRow := true

		// clipper length is divided by the number of columns and is used to
		// tell the ListClipper how much data to expect
		//
		// we add numColumns to make sure we include the last line which may be
		// an incomplete row and would otherwise be missed out of the clipper
		//
		// we also make sure we adjust for the number of leading columns
		//
		// note that this strategy requires a check that offset does not exceed
		// the actual length of the data
		clipperLen := len(data) + numColumns + leadingColumns - 1

		// offset and address will be increased as we draw each column

		imgui.ListClipperAll(clipperLen/numColumns, func(i int) {
			idx := (i * numColumns) - leadingColumns
			addr := origin + uint32(idx)

			imgui.TableNextRow()
			imgui.TableNextColumn()
			imgui.AlignTextToFramePadding()
			imgui.Text(fmt.Sprintf("%08x-", addr/16)[columnCrop+1:])

			// column limit for row changes depending on the requirements
			// of the first row
			columnLimitForRow := numColumns

			// add blank columns to first row as necessary
			if firstRow {
				for range leadingColumns {
					imgui.TableNextColumn()
					idx++
					addr++
				}
				columnLimitForRow -= leadingColumns
				firstRow = false
			}

			for j := 0; j < columnLimitForRow; j++ {
				// check that offset hasn't gone beyond the end of data
				if idx >= len(data) {
					break
				}

				imgui.TableNextColumn()

				if before != nil {
					before(idx)
				}

				// editable byte
				b := data[idx]

				s := fmt.Sprintf("%02x", b)
				if imguiHexInput(fmt.Sprintf("%s##%08x", id, addr), 2, &s) {
					if v, err := strconv.ParseUint(s, 16, 8); err == nil {
						commit(idx, uint8(v))
					}
				}

				if after != nil {
					after(idx)
				}

				// advance offset and addr by one
				idx++
				addr++
			}
		})
	}
}

// draw grid of bytes with automated diff highlighting and tooltip handling
//
// see drawByteGrid() for more flexible alternative.
func (img *SdlImgui) drawByteGridSimple(id string, data []uint8, diff []uint8, diffCol imgui.Vec4, origin uint32,
	commit func(int, uint8)) {

	var a uint8
	var b uint8

	before := func(idx int) {
		// editable byte
		a = data[idx]

		// compare current RAM value with value in comparison snapshot and use
		// highlight color if it is different
		b = a
		if diff != nil {
			b = diff[idx]
		}
		if a != b {
			imgui.PushStyleColor(imgui.StyleColorFrameBg, diffCol)
		}
	}

	after := func(idx int) {
		if a != b {
			img.imguiTooltip(func() {
				imguiColorLabelSimple(fmt.Sprintf("%02x %c %02x", b, fonts.ByteChange, a), diffCol)
			}, true)
		}

		// undo any color changes
		if a != b {
			imgui.PopStyleColor()
		}
	}

	drawByteGrid(id, data, origin, before, after, commit)
}
