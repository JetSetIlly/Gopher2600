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
	"github.com/jetsetilly/gopher2600/coprocessor/developer"
	"github.com/jetsetilly/gopher2600/gui/fonts"
)

const winCoProcIllegalAccessID = "Coprocessor Illegal Accesses"
const winCoProcIllegalAccessMenu = "Illegal Accesses"

type winCoProcIllegalAccess struct {
	debuggerWin

	img *SdlImgui

	showSrcInTooltip bool
	optionsHeight    float32
}

func newWinCoProcIllegalAccess(img *SdlImgui) (window, error) {
	win := &winCoProcIllegalAccess{
		img:              img,
		showSrcInTooltip: true,
	}
	return win, nil
}

func (win *winCoProcIllegalAccess) init() {
}

func (win *winCoProcIllegalAccess) id() string {
	return winCoProcIllegalAccessID
}

func (win *winCoProcIllegalAccess) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	if !win.img.lz.Cart.HasCoProcBus {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{982, 77}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{520, 390}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{400, 300}, imgui.Vec2{551, 1000})

	title := fmt.Sprintf("%s %s", win.img.lz.Cart.CoProcID, winCoProcIllegalAccessID)
	if imgui.BeginV(win.debuggerID(title), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCoProcIllegalAccess) draw() {
	// hasStackCollision to decide whether to issue warning in footer
	hasStackCollision := false

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

		// note HasStackCollision for later comparison
		hasStackCollision = ill.HasStackCollision

		const numColumns = 3

		flgs := imgui.TableFlagsScrollY
		flgs |= imgui.TableFlagsSizingStretchProp
		flgs |= imgui.TableFlagsResizable

		imgui.BeginTableV("##coprocIllegalAccessTable", numColumns, flgs, imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.optionsHeight}, 0.0)

		// setup columns. the labelling column 2 depends on whether the coprocessor
		// development instance has source available to it
		width := imgui.ContentRegionAvail().X
		imgui.TableSetupColumnV("Event", imgui.TableColumnFlagsNone, width*0.30, 0)
		imgui.TableSetupColumnV("Address", imgui.TableColumnFlagsNone, width*0.20, 1)
		if win.img.dbg.CoProcDev.HasSource() {
			imgui.TableSetupColumnV("Function", imgui.TableColumnFlagsNone, width*0.45, 2)
		} else {
			imgui.TableSetupColumnV("PC Address", imgui.TableColumnFlagsNone, width*0.45, 2)
		}

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
			win.img.imguiTooltip(func() {
				imgui.Text("Executing PC:")
				imgui.SameLine()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcIllegalAccessAddress)
				imgui.Text(fmt.Sprintf("%08x", lg.PC))
				imgui.PopStyleColor()

				if lg.IsNullAccess {
					imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcIllegalAccessNotes)
					imgui.Text("likely null pointer dereference")
					imgui.PopStyleColor()
				}

				imgui.Text("Frequency:")
				imgui.SameLine()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcIllegalAccessFrequency)
				imgui.Text(fmt.Sprintf("%d", lg.Count))
				imgui.PopStyleColor()

				if win.showSrcInTooltip {
					if !lg.SrcLine.IsStub() {
						imgui.Spacing()
						imgui.Separator()
						imgui.Spacing()

						win.img.drawFilenameAndLineNumber(lg.SrcLine.File.Filename, lg.SrcLine.LineNumber, -1)

						imgui.Spacing()
						imgui.Separator()
						imgui.Spacing()
						win.img.drawSourceLine(lg.SrcLine, true)
						if len(lg.SrcLine.Instruction) > 0 {
							imgui.Spacing()
							imgui.Separator()
							imgui.Spacing()
							win.img.drawDisasmForCoProc(lg.SrcLine.Instruction, lg.SrcLine, false)
						}
					} else {
						imgui.Spacing()
						imgui.Separator()
						imgui.Spacing()
						imgui.Text("No source for this instruction")
					}
				}
			}, true)

			// open source window on click
			if imgui.IsItemClicked() && !lg.SrcLine.IsStub() {
				srcWin := win.img.wm.debuggerWindows[winCoProcSourceID].(*winCoProcSource)
				srcWin.gotoSourceLine(lg.SrcLine)
			}

			imgui.TableNextColumn()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcIllegalAccessAddress)
			imgui.Text(fmt.Sprintf("%08x", lg.AccessAddr))
			imgui.PopStyleColor()

			imgui.TableNextColumn()
			if win.img.dbg.CoProcDev.HasSource() {
				if lg.SrcLine != nil {
					imgui.Text(lg.SrcLine.Function.Name)
				} else {
					// in case function name cannot be found
					imgui.Text("-")
				}
			} else {
				// show PC address if there is no source available
				imgui.Text(fmt.Sprintf("%#08x", lg.PC))
			}
		}

		imgui.EndTable()

		// options toolbar at foot of window
		win.optionsHeight = imguiMeasureHeight(func() {
			imgui.Separator()
			imgui.Spacing()

			if win.img.dbg.CoProcDev.HasSource() {
				win.img.dbg.CoProcDev.BorrowSource(func(src *developer.Source) {
					if src == nil {
						return
					}
				})

				imgui.Checkbox("Show Source in Tooltip", &win.showSrcInTooltip)
			} else {
				imgui.Text("no source available")
			}

			if hasStackCollision {
				imgui.SameLineV(0, 20)
				imgui.BeginGroup()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.Warning)
				imgui.AlignTextToFramePadding()
				imgui.Text(fmt.Sprintf(" %c", fonts.Warning))
				imgui.PopStyleColor()
				imgui.EndGroup()
				imgui.SameLine()
				imgui.Text("Stack collision detected")
				win.img.imguiTooltip(func() {
					imgui.Text("Memory access is unreliable after a stack collision. Illegal")
					imgui.Text("accesses are no longer being logged.")
				}, true)
			}
		})
	})
}
