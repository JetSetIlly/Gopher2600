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

	"github.com/jetsetilly/imgui-go/v5"
)

type byteGridConfig struct {
	origin uint32
	data   []uint8
	commit func(idx int, value uint8)

	// the remaining fields can be nil

	// if diff is provided then the byte entry is highlighted to illustrate whether the byte has
	// changed since the comparision point
	diff []uint8

	// hook to be called before the drawing of the byte entry
	before func(idx int)

	// hook to be called after the drawing of the byte entry. returns true if the function drew
	// something to the tooltip. this will affect the drawing of a standard tooltip
	after func(idx int) bool

	// hooks to be called if the text that titles each row is not standard
	rowTitle func(addr uint32)
}

func (img *SdlImgui) drawByteGrid(id string, cfg byteGridConfig) {
	// the number of characters in an address. used to decide on the width of the first column in
	// the table
	addressLength := len(fmt.Sprintf("%x", cfg.origin+uint32(len(cfg.data)-1)))

	spacing := imgui.Vec2{X: 0.5, Y: 0.5}
	imgui.PushStyleVarVec2(imgui.StyleVarCellPadding, spacing)
	defer imgui.PopStyleVar()

	const numColumns = 16

	flgs := imgui.TableFlagsScrollY
	flgs |= imgui.TableFlagsSizingFixedFit

	if imgui.BeginTableV(id, numColumns+1, flgs, imgui.Vec2{}, 0.0) {
		// in some situations we will return early from the drawByteGrid()
		// function so we want to make sure that EndTable() is called
		defer imgui.EndTable()

		imgui.TableSetupScrollFreeze(0, 1)

		// set up columns
		width := imguiTextWidth(addressLength - 1)
		imgui.TableSetupColumnV(fmt.Sprintf("%p_column0", cfg.data), imgui.TableColumnFlagsNone, width, 0)
		width = imguiTextWidth(2)
		for i := 1; i < numColumns+1; i++ {
			imgui.TableSetupColumnV(fmt.Sprintf("%p_column%d", cfg.data, i), imgui.TableColumnFlagsNone, width, 0)
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
		leadingColumns := int(cfg.origin % numColumns)

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
		clipperLen := len(cfg.data) + numColumns + leadingColumns - 1

		// offset and address will be increased as we draw each column

		imgui.ListClipperAll(clipperLen/numColumns, func(i int) {
			idx := (i * numColumns) - leadingColumns
			addr := cfg.origin + uint32(idx)

			imgui.TableNextRow()
			imgui.TableNextColumn()
			imgui.AlignTextToFramePadding()
			if cfg.rowTitle == nil {
				imgui.Textf("%x-", addr/16)
			} else {
				cfg.rowTitle(addr)
			}

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
				if idx >= len(cfg.data) {
					break
				}

				imgui.TableNextColumn()

				// editable byte
				b := cfg.data[idx]

				// compare current RAM value with value in comparison snapshot and use
				// highlight color if it is different
				var highlight bool
				if cfg.diff != nil {
					highlight = b != cfg.diff[idx]
					if highlight {
						imgui.PushStyleColor(imgui.StyleColorFrameBg, img.cols.ValueDiff)
					}
				}

				if cfg.before != nil {
					cfg.before(idx)
				}

				s := fmt.Sprintf("%02x", b)
				if imguiHexInput(fmt.Sprintf("%s##%08x", id, addr), 2, &s) {
					if v, err := strconv.ParseUint(s, 16, 8); err == nil {
						cfg.commit(idx, uint8(v))
					}
				}

				var tooltipDrawn bool
				if cfg.after != nil {
					tooltipDrawn = cfg.after(idx)
				}

				if highlight {
					imgui.PopStyleColor()
					img.imguiTooltip(func() {
						if tooltipDrawn {
							imgui.Spacing()
							imgui.Separator()
							imgui.Spacing()
						}
						imguiColorLabel(fmt.Sprintf("previously $%02x, now $%02x", cfg.diff[idx], b), img.cols.ValueDiff)
					}, true)
				}

				// advance offset and addr by one
				idx++
				addr++
			}
		})
	}
}
