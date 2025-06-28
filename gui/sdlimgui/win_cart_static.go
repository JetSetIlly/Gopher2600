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

	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/imgui-go/v5"
)

const winCartStaticID = "Static Areas"
const winCartStaticPopupID = "winCartStaticPopupID"

type winCartStatic struct {
	debuggerWin

	img *SdlImgui

	// the name of the selected static memory area
	selectedArea string
}

func newWinCartStatic(img *SdlImgui) (window, error) {
	win := &winCartStatic{img: img}

	return win, nil
}

func (win *winCartStatic) init() {
}

func (win *winCartStatic) id() string {
	return winCartStaticID
}

func (win *winCartStatic) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	// do not open window if there is no cartridge static bus available
	bus := win.img.cache.VCS.Mem.Cart.GetStaticBus()
	if bus == nil {
		return false
	}
	static := bus.GetStatic()

	imgui.SetNextWindowPosV(imgui.Vec2{X: 117, Y: 248}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 468, Y: 552}, imgui.ConditionFirstUseEver)
	win.img.setReasonableWindowConstraints()

	title := fmt.Sprintf("%s %s", win.img.cache.VCS.Mem.Cart.ID(), winCartStaticID)
	if imgui.BeginV(win.debuggerID(title), &win.debuggerOpen, imgui.WindowFlagsHorizontalScrollbar) {
		win.draw(static)

		// popup menu on right mouse button
		if imgui.IsItemHovered() && imgui.IsMouseDown(1) {
			imgui.OpenPopup(winCartStaticPopupID)
		}

		if imgui.BeginPopup(winCartStaticPopupID) {
			imgui.Text("Dump to file")
			imgui.Spacing()
			imgui.Separator()
			imgui.Spacing()
			if imgui.Selectable(fmt.Sprintf("Current area '%s'", win.selectedArea)) {
				win.img.term.pushCommand(fmt.Sprintf("COPROC MEM DUMP %s", win.selectedArea))
			}
			if imgui.Selectable("All areas") {
				win.img.term.pushCommand("COPROC MEM DUMP")
			}
			imgui.EndPopup()
		}
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCartStatic) draw(static mapper.CartStatic) {
	imgui.BeginGroup()
	defer imgui.EndGroup()

	// get comparison data. assuming that there is such a thing and that it's
	// safe to get StaticData from.
	compStatic := win.img.cache.Rewind.Comparison.State.VCS.Mem.Cart.GetStaticBus().GetStatic()

	// make a note of cell padding value. this changes for the duration of
	// drawByteGrid() but we want the default value for when we call
	// drawVariableTooltip(). there is a table in that tooltip which will be
	// affected by the CellPadding value
	cellPadding := imgui.CurrentStyle().CellPadding()

	imgui.BeginTabBar("")
	for _, seg := range static.Segments() {
		// skip any segments that are empty for whatever reason
		if seg.Origin == seg.Memtop {
			continue
		}

		if imgui.BeginTabItemV(seg.Name, nil, 0) {
			win.selectedArea = seg.Name

			imgui.Spacing()
			imgui.Text("Origin:")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesAddress)
			imgui.Text(fmt.Sprintf("%08x", seg.Origin))
			imgui.PopStyleColor()
			imgui.SameLineV(0, 20)
			imgui.Text("Memtop:")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.CoProcVariablesAddress)
			imgui.Text(fmt.Sprintf("%08x", seg.Memtop))
			imgui.PopStyleColor()

			imgui.Spacing()
			imgui.Spacing()

			imgui.BeginChildV("cartstatic", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight()}, false, 0)

			currStatic := static
			currData, ok := currStatic.Reference(seg.Name)

			if ok {
				win.img.dbg.CoProcDev.BorrowSource(func(src *dwarf.Source) {
					// the borrowed source instance is used in the after()
					// function to help decorate the memory cells in the table.
					// a check for nil is done then

					compData, ok := compStatic.Reference(seg.Name)
					if ok {
						// take copy of seg.Name because we'll be accessing it in a PushFunction() below
						segname := seg.Name

						// pos is retreived in before() and used in after()
						var pos imgui.Vec2

						// number of colors to pop in afer()
						popColor := 0

						before := func(idx int) {
							pos = imgui.CursorScreenPos()

							// difference colour
							a := currData[idx]
							b := compData[idx]
							if a != b {
								imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.ValueDiff)
								popColor++
							}
						}

						after := func(idx int) {
							imgui.PopStyleColorV(popColor)
							popColor = 0

							// if no source is available then there is nothing
							// more to do in the after() function
							if src == nil {
								return
							}

							// idx is based on original values of type uint16 so the type conversion is safe
							addr := seg.Origin + uint32(idx)

							dl := imgui.WindowDrawList()
							if varb, ok := src.GlobalsByAddress[uint64(addr)]; ok {
								sz := imgui.FontSize() * 0.4
								pos.X += 1.0
								pos.Y += 1.0
								p1 := pos
								p1.Y += sz
								p2 := pos
								p2.X += sz
								dl.AddTriangleFilled(pos, p1, p2, imgui.PackedColorFromVec4(win.img.cols.ValueSymbol))

								win.img.imguiTooltip(func() {
									var value uint32
									currValue := "??"
									compValue := "??"

									switch varb.Type.Size {
									case 4:
										if d, ok := currStatic.Read32bit(addr); ok {
											value = d
											currValue = fmt.Sprintf("%08x", d)
										}
										if d, ok := compStatic.Read32bit(addr); ok {
											compValue = fmt.Sprintf("%08x", d)
										}
									case 2:
										if d, ok := currStatic.Read16bit(addr); ok {
											value = uint32(d)
											currValue = fmt.Sprintf("%04x", d)
										}
										if d, ok := compStatic.Read16bit(addr); ok {
											compValue = fmt.Sprintf("%04x", d)
										}
									default:
										if d, ok := currStatic.Read8bit(addr); ok {
											value = uint32(d)
											currValue = fmt.Sprintf("%02x", d)
										}
										if d, ok := compStatic.Read8bit(addr); ok {
											compValue = fmt.Sprintf("%02x", d)
										}
									}

									// wrap drawVariableTooltip() in a cellpadding style. see comment for
									// cellPadding declartion above
									imgui.PushStyleVarVec2(imgui.StyleVarCellPadding, cellPadding)
									drawVariableTooltip(varb, value, win.img.cols)
									imgui.PopStyleVar()

									if currValue != compValue {
										imgui.Spacing()
										imgui.Separator()
										imgui.Spacing()
										imguiColorLabelSimple(fmt.Sprintf("%s %c %s", compValue, fonts.ByteChange, currValue), win.img.cols.ValueDiff)
									}
								}, true)
							} else {
								a := compData[idx]
								b := currData[idx]
								if a != b {
									win.img.imguiTooltip(func() {
										imguiColorLabelSimple(fmt.Sprintf("%02x %c %02x", a, fonts.ByteChange, b), win.img.cols.ValueDiff)
									}, true)
								}
							}
						}

						commit := func(idx int, data uint8) {
							win.img.dbg.PushFunction(func() {
								win.img.dbg.VCS().Mem.Cart.GetStaticBus().PutStatic(segname, idx, data)
							})
						}

						drawByteGrid(fmt.Sprintf("##cartStatic##%s", segname), currData, seg.Origin, before, after, commit)
					}
				})
			}

			imgui.EndChild()

			imgui.EndTabItem()
		}
	}
	imgui.EndTabBar()
}
