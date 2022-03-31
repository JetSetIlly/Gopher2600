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
	"strings"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/coprocessor/developer"
	"github.com/jetsetilly/gopher2600/gui/fonts"
)

// in this case of the coprocessor disassmebly window the actual window title
// is prepended with the actual coprocessor ID (eg. ARM7TDMI). The ID constant
// below is used in the normal way however.

const winCoProcIllegalAccessID = "Coprocessor Illegal Accesses"
const winCoProcIllegalAccessMenu = "Illegal Accesses"

type winCoProcIllegalAccess struct {
	img           *SdlImgui
	open          bool
	showSrc       bool
	optionsHeight float32
}

func newWinCoProcIllegalAccess(img *SdlImgui) (window, error) {
	win := &winCoProcIllegalAccess{
		img:     img,
		showSrc: true,
	}
	return win, nil
}

func (win *winCoProcIllegalAccess) init() {
}

func (win *winCoProcIllegalAccess) id() string {
	return winCoProcIllegalAccessID
}

func (win *winCoProcIllegalAccess) isOpen() bool {
	return win.open
}

func (win *winCoProcIllegalAccess) setOpen(open bool) {
	win.open = open
}

func (win *winCoProcIllegalAccess) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.HasCoProcBus || win.img.dbg.CoProcDev == nil {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{1051, 89}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{520, 390}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{400, 300}, imgui.Vec2{551, 1000})

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.CoProcID, winCoProcIllegalAccessID)
	imgui.BeginV(title, &win.open, imgui.WindowFlagsNone)
	defer imgui.End()

	// safely iterate over top execution information
	win.img.dbg.CoProcDev.BorrowIllegalAccess(func(ill *developer.IllegalAccess) {
		if ill == nil {
			imgui.Text("No illegal accesses")
			return
		}

		if len(ill.Log) == 0 {
			imgui.Text("No illegal accesses")
			return
		}

		imgui.BeginChildV("##coprocIllegalAccessMain", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.optionsHeight}, false, 0)

		const numColumns = 4
		imgui.BeginTableV("##coprocIllegalAccessTable", numColumns, imgui.TableFlagsSizingFixedFit, imgui.Vec2{}, 0.0)

		// first column is a dummy column so that Selectable (span all columns) works correctly
		width := imgui.ContentRegionAvail().X
		imgui.TableSetupColumnV("Event", imgui.TableColumnFlagsNone, width*0.20, 1)
		imgui.TableSetupColumnV("Address", imgui.TableColumnFlagsNone, width*0.20, 3)
		imgui.TableSetupColumnV("Function", imgui.TableColumnFlagsNone, width*0.45, 3)
		imgui.TableSetupColumnV("Count", imgui.TableColumnFlagsNone, width*0.10, 3)

		imgui.Spacing()
		imgui.TableHeadersRow()

		for i := 0; i < len(ill.Log); i++ {
			imgui.TableNextRow()
			lg := ill.Log[i]

			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorHeaderHovered, win.img.cols.CoProcSourceHover)
			imgui.PushStyleColor(imgui.StyleColorHeaderActive, win.img.cols.CoProcSourceHover)
			imgui.SelectableV(lg.Event, false, imgui.SelectableFlagsSpanAllColumns, imgui.Vec2{0, 0})
			imgui.PopStyleColorV(2)

			// source on tooltip
			if win.showSrc && lg.SrcLine != nil {
				imguiTooltip(func() {
					imgui.Text(lg.SrcLine.File.Filename)
					imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcSourceLineNumber)
					imgui.Text(fmt.Sprintf("Line: %d", lg.SrcLine.LineNumber))
					imgui.PopStyleColor()
					imgui.Spacing()
					imgui.Separator()
					imgui.Spacing()
					imgui.Text(strings.TrimSpace(lg.SrcLine.PlainContent))
				}, true)
			}

			// open source window on click
			if imgui.IsItemClicked() && lg.SrcLine != nil {
				srcWin := win.img.wm.windows[winCoProcSourceID].(*winCoProcSource)
				srcWin.gotoSourceLine(lg.SrcLine)
			}

			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcIllegalAccessAddress)
			imgui.Text(fmt.Sprintf("%#08x", lg.AccessAddr))
			imgui.PopStyleColor()

			imgui.TableNextColumn()
			if lg.SrcLine != nil {
				imgui.Text(lg.SrcLine.Function.Name)
			}

			imgui.TableNextColumn()
			imgui.Text(fmt.Sprintf("%d", lg.Count))
		}

		imgui.EndTable()
		imgui.EndChild()

		if win.img.dbg.CoProcDev.HasSource() {
			// options toolbar at foot of window
			win.optionsHeight = imguiMeasureHeight(func() {
				imgui.Separator()
				imgui.Spacing()

				win.img.dbg.CoProcDev.BorrowSource(func(src *developer.Source) {
					if src == nil {
						return
					}

					if src.UnsupportedOptimisation != "" {
						imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.Warning)
						imgui.AlignTextToFramePadding()
						imgui.Text(fmt.Sprintf(" %c", fonts.Warning))
						imgui.PopStyleColor()
						imguiTooltip(func() {
							imgui.Text(src.UnsupportedOptimisation)
							imgui.Text("illegal access analysis may be misleading")
						}, true)
						imgui.SameLineV(0, 20)
					}
				})

				imgui.Checkbox("Show Source in Tooltip", &win.showSrc)
			})
		} else {
			win.optionsHeight = 0.0
		}
	})
}
