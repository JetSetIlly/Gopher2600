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

	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
	"github.com/jetsetilly/imgui-go/v5"
)

func (img *SdlImgui) drawRegistersForCoProcDisasmEntry(id string, e arm.DisasmEntry) {
	var flgs imgui.TableFlags
	flgs = imgui.TableFlagsBordersInnerV
	if imgui.BeginTableV(fmt.Sprintf("%s##coprocRegisters", id), 2, flgs, imgui.Vec2{}, 0.0) {
		for i := 0; i < len(e.Registers); i += 2 {
			imgui.TableNextRow()
			imgui.TableNextColumn()
			imgui.Text(fmt.Sprintf("R%02d: %08x", i, e.Registers[i]))
			imgui.TableNextColumn()
			imgui.Text(fmt.Sprintf("R%02d: %08x", i+1, e.Registers[i+1]))
		}
		imgui.EndTable()
	}
}

func (img *SdlImgui) drawDisasmForCoProc(id string, disasm []*dwarf.SourceInstruction, ln *dwarf.SourceLine,
	indicator bool, address uint32, tooltip bool) {

	flgs := imgui.TableFlagsSizingFixedFit
	if imgui.BeginTableV(fmt.Sprintf("%s##disasmTable", id), 4, flgs, imgui.Vec2{}, 1.0) {
		yldLine := 0
		for i := range disasm {
			d := disasm[i]
			if d.Addr == address {
				yldLine = i
				break
			}
		}

		// find window limits
		var start, end int
		start = 0
		end = len(disasm)

		// the maximum the number of lines to show in the 'window' depends on the tooltip flag
		if tooltip {
			const windowSize = 10
			start = max(yldLine-(windowSize/2), 0)
			end = min(start+windowSize, len(disasm))
		}

		// add prelude elipses if the 'window' is not placed at the beginning of the list
		if tooltip && start > 0 {
			imgui.TableNextRow()
			imgui.TableNextColumn()
			imgui.TableNextColumn()
			imgui.TableNextColumn()
			imgui.TableNextColumn()
			imgui.Text("...")
		}

		for i := start; i < end; i++ {
			d := disasm[i]

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
			if indicator {
				if d.Addr == address {
					imgui.PushStyleColor(imgui.StyleColorText, img.cols.CoProcSourceYield)
					imgui.Text(string(fonts.TermPrompt))
					imgui.PopStyleColor()
				}
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
		}

		// add epilogue elipses if the 'window' does not reach the end of the list
		if tooltip && end < len(disasm) {
			imgui.Text("...")
		}
		imgui.EndTable()
	}
}

// display source line with syntax highlighting.
func (img *SdlImgui) drawSourceLine(ln *dwarf.SourceLine, tight bool) {
	for _, fr := range ln.Fragments {
		s := fr.Content
		if tight {
			s = strings.TrimSpace(s)
		}

		switch fr.Type {
		case dwarf.FragmentCode:
			imgui.Text(s)
		case dwarf.FragmentComment:
			imgui.PushStyleColor(imgui.StyleColorText, img.cols.CoProcSourceComment)
			imgui.Text(s)
			imgui.PopStyleColor()
		case dwarf.FragmentStringLiteral:
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

// add a warning symbol with tooltip that indicates potential problems when interpreting the
// coprocessor source information (eg. profiling data)
//
// colour of symbol is red if 'immediate mode' is enabled and yellow if an optimised compilation is
// detected. symbol is not present if neither case is true
func (img *SdlImgui) drawCoprocSourceWarning(src *dwarf.Source) {
	immediate := img.dbg.VCS().Env.Prefs.Cartridge.ARM.Immediate.Get().(bool)
	warning := immediate || src.Optimisation
	if warning {
		if immediate {
			imgui.PushStyleColor(imgui.StyleColorText, img.cols.Danger)
		} else {
			imgui.PushStyleColor(imgui.StyleColorText, img.cols.Warning)
		}
		imgui.Text(string(fonts.Warning))
		imgui.PopStyleColor()

		if immediate {
			img.imguiTooltipSimple("Emulation in 'immediate ARM execution' mode. Profiling unavailable")
		} else if src.Optimisation {
			img.imguiTooltipSimple("Source compiled with optimisation. Some profiling results may be misleading")
		}
	}
}
