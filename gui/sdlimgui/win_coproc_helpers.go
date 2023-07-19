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

func (img *SdlImgui) drawDisasmForCoProc(disasm []*developer.SourceInstruction, ln *developer.SourceLine,
	multiline bool, showYield bool, yldAddress uint32) {

	imgui.BeginTable("##disasmTable", 4)

	const maxDisasmLines = 10
	numLines := 0

	// draw disassembly, colouring the text according to whether the disassembly entry
	// is associated with the current line (ie. the one the mouse is over)
	for _, d := range disasm {
		imgui.TableNextRow()

		imgui.TableNextColumn()
		if d.Line.LineNumber == ln.LineNumber {
			imgui.PushStyleColor(imgui.StyleColorText, img.cols.CoProcSourceDisasmAddr)
		} else {
			imgui.PushStyleColor(imgui.StyleColorText, img.cols.CoProcSourceDisasmAddrFade)
		}
		imgui.Text(fmt.Sprintf("%08x", d.Addr))

		imgui.PopStyleColor()

		imgui.TableNextColumn()
		if showYield {

			// simple way of making sure the yield column doesn't change width
			// is to always print the icon but to use an the window backtround
			// colour if the icon is to be invisible
			if d.Addr == yldAddress {
				imgui.PushStyleColor(imgui.StyleColorText, img.cols.CoProcSourceYield)
			} else {
				imgui.PushStyleColor(imgui.StyleColorText, img.cols.WindowBg)
			}

			imgui.Text(string(fonts.Breakpoint))
			imgui.PopStyleColor()
		}

		imgui.TableNextColumn()
		if d.Line.LineNumber == ln.LineNumber {
			imgui.PushStyleColor(imgui.StyleColorText, img.cols.CoProcSourceDisasmOpcode)
		} else {
			imgui.PushStyleColor(imgui.StyleColorText, img.cols.CoProcSourceDisasmOpcodeFade)
		}
		imgui.Text(d.Opcode())
		imgui.PopStyleColor()

		imgui.TableNextColumn()
		if d.Line.LineNumber == ln.LineNumber {
			imgui.PushStyleColor(imgui.StyleColorText, img.cols.CoProcSourceDisasm)
		} else {
			imgui.PushStyleColor(imgui.StyleColorText, img.cols.CoProcSourceDisasmFade)
		}
		imgui.Text(d.Disasm.String())
		imgui.PopStyleColor()

		// limit number of lines unless multiline is true
		numLines++
		if numLines >= maxDisasmLines && !multiline {
			if len(disasm) > maxDisasmLines {
				imgui.Text("...")
				break // for loop
			}
		}
	}

	imgui.EndTable()
}

// display source line with syntax highlighting.
func (img *SdlImgui) drawSourceLine(ln *developer.SourceLine, tight bool) {
	for _, fr := range ln.Fragments {
		s := fr.Content
		if tight {
			s = strings.TrimSpace(s)
		}

		switch fr.Type {
		case developer.FragmentCode:
			imgui.Text(s)
		case developer.FragmentComment:
			imgui.PushStyleColor(imgui.StyleColorText, img.cols.CoProcSourceComment)
			imgui.Text(s)
			imgui.PopStyleColor()
		case developer.FragmentStringLiteral:
			imgui.PushStyleColor(imgui.StyleColorText, img.cols.CoProcSourceStringLiteral)
			imgui.Text(s)
			imgui.PopStyleColor()
		}

		if tight {
			imgui.SameLine()
		} else {
			imgui.SameLineV(0, 0)
		}
	}

	// undo last call to SameLine() with a call to Spacing()
	imgui.Spacing()
}

func (img *SdlImgui) drawFilenameAndLineNumber(filename string, lineStart int, lineEnd int) {
	imgui.Text(filename)
	imgui.PushStyleColor(imgui.StyleColorText, img.cols.CoProcSourceLineNumber)
	if lineEnd < 0 {
		imgui.Text(fmt.Sprintf("Line: %d", lineStart))
	} else {
		imgui.Text(fmt.Sprintf("Lines: %d - %d", lineStart, lineEnd))
	}
	imgui.PopStyleColor()
}
