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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlimgui

import (
	"github.com/jetsetilly/gopher2600/television/colors"

	"github.com/inkyblackness/imgui-go/v2"
)

type vec4Palette []imgui.Vec4
type packedPalette []imgui.PackedColor

// imguiColors defines all the colors used by the GUI
type imguiColors struct {
	// default colors
	MenuBarBg     imgui.Vec4
	WindowBg      imgui.Vec4
	TitleBg       imgui.Vec4
	TitleBgActive imgui.Vec4
	Border        imgui.Vec4

	// the color to draw the TV Screen window border when mouse is captured
	CapturedScreenTitle  imgui.Vec4
	CapturedScreenBorder imgui.Vec4

	// CPU status register buttons
	CPUStatusOn  imgui.Vec4
	CPUStatusOff imgui.Vec4
	CPUFlgRdyOn  imgui.Vec4
	CPUFlgRdyOff imgui.Vec4

	// control window buttons
	ControlRun         imgui.Vec4
	ControlRunHovered  imgui.Vec4
	ControlRunActive   imgui.Vec4
	ControlHalt        imgui.Vec4
	ControlHaltHovered imgui.Vec4
	ControlHaltActive  imgui.Vec4

	// disassembly entry columns
	DisasmAddress  imgui.Vec4
	DisasmMnemonic imgui.Vec4
	DisasmOperand  imgui.Vec4
	DisasmCycles   imgui.Vec4
	DisasmNotes    imgui.Vec4

	// disassembly other
	DisasmCurrHighlight imgui.Vec4
	DisasmBreakAddress  imgui.Vec4
	DisasmBreakOther    imgui.Vec4

	// audio oscilloscope
	AudioOscBg   imgui.Vec4
	AudioOscLine imgui.Vec4

	// tia window
	IdxPointer imgui.Vec4

	// terminal
	TermBackground           imgui.Vec4
	TermStyleInput           imgui.Vec4
	TermStyleHelp            imgui.Vec4
	TermStylePromptCPUStep   imgui.Vec4
	TermStylePromptVideoStep imgui.Vec4
	TermStylePromptConfirm   imgui.Vec4
	TermStyleFeedback        imgui.Vec4
	TermStyleCPUStep         imgui.Vec4
	TermStyleVideoStep       imgui.Vec4
	TermStyleInstrument      imgui.Vec4
	TermStyleError           imgui.Vec4

	vec4PaletteNTSC   vec4Palette
	vec4PalettePAL    vec4Palette
	vec4PaletteAlt    vec4Palette
	packedPaletteNTSC packedPalette
	packedPalettePAL  packedPalette
	packedPaletteAlt  packedPalette
}

func newColors() *imguiColors {
	cols := imguiColors{
		// default colors
		MenuBarBg:     imgui.Vec4{0.075, 0.08, 0.09, 1.0},
		WindowBg:      imgui.Vec4{0.075, 0.08, 0.09, 1.0},
		TitleBg:       imgui.Vec4{0.075, 0.08, 0.09, 1.0},
		TitleBgActive: imgui.Vec4{0.16, 0.29, 0.48, 1.0},
		Border:        imgui.Vec4{0.14, 0.14, 0.29, 1.0},

		// deferring CapturedScreenTitle & CapturedScreenBorder

		// CPU status register buttons
		CPUStatusOn:  imgui.Vec4{0.8, 0.6, 0.2, 1.0},
		CPUStatusOff: imgui.Vec4{0.7, 0.5, 0.1, 1.0},
		CPUFlgRdyOn:  imgui.Vec4{0.3, 0.6, 0.3, 1.0},
		CPUFlgRdyOff: imgui.Vec4{0.6, 0.3, 0.3, 1.0},

		// control window buttons
		ControlRun:         imgui.Vec4{0.3, 0.6, 0.3, 1.0},
		ControlRunHovered:  imgui.Vec4{0.3, 0.65, 0.3, 1.0},
		ControlRunActive:   imgui.Vec4{0.3, 0.65, 0.3, 1.0},
		ControlHalt:        imgui.Vec4{0.6, 0.3, 0.3, 1.0},
		ControlHaltHovered: imgui.Vec4{0.65, 0.3, 0.3, 1.0},
		ControlHaltActive:  imgui.Vec4{0.65, 0.3, 0.3, 1.0},

		// disassembly entry columns
		DisasmAddress:  imgui.Vec4{0.8, 0.4, 0.4, 1.0},
		DisasmMnemonic: imgui.Vec4{0.4, 0.4, 0.8, 1.0},
		DisasmOperand:  imgui.Vec4{0.8, 0.8, 0.3, 1.0},
		DisasmCycles:   imgui.Vec4{0.8, 0.8, 0.8, 1.0},
		DisasmNotes:    imgui.Vec4{0.8, 0.8, 0.8, 1.0},

		// disassembly other
		DisasmCurrHighlight: imgui.Vec4{1.0, 1.0, 1.0, 0.1},
		// deferring DisasmBreakAddress & DisasmBreakOther

		// audio oscilloscope
		AudioOscBg:   imgui.Vec4{0.21, 0.29, 0.23, 1.0},
		AudioOscLine: imgui.Vec4{0.10, 0.97, 0.29, 1.0},

		// tia
		IdxPointer: imgui.Vec4{0.8, 0.8, 0.8, 1.0},

		// terminal
		TermBackground:           imgui.Vec4{0.1, 0.1, 0.2, 0.9},
		TermStyleInput:           imgui.Vec4{0.8, 0.8, 0.8, 1.0},
		TermStyleHelp:            imgui.Vec4{1.0, 1.0, 1.0, 1.0},
		TermStylePromptCPUStep:   imgui.Vec4{1.0, 1.0, 1.0, 1.0},
		TermStylePromptVideoStep: imgui.Vec4{0.8, 0.8, 0.8, 1.0},
		TermStylePromptConfirm:   imgui.Vec4{0.1, 0.4, 0.9, 1.0},
		TermStyleFeedback:        imgui.Vec4{1.0, 1.0, 1.0, 1.0},
		TermStyleCPUStep:         imgui.Vec4{0.9, 0.9, 0.5, 1.0},
		TermStyleVideoStep:       imgui.Vec4{0.7, 0.7, 0.3, 1.0},
		TermStyleInstrument:      imgui.Vec4{0.1, 0.95, 0.9, 1.0},
		TermStyleError:           imgui.Vec4{0.8, 0.3, 0.3, 1.0},
	}

	// we deferred setting of some colours. set them now.
	cols.CapturedScreenTitle = cols.TitleBgActive
	cols.CapturedScreenBorder = cols.TitleBgActive
	cols.DisasmBreakAddress = cols.DisasmAddress
	cols.DisasmBreakOther = cols.DisasmMnemonic

	// set default colors
	style := imgui.CurrentStyle()
	style.SetColor(imgui.StyleColorMenuBarBg, cols.MenuBarBg)
	style.SetColor(imgui.StyleColorWindowBg, cols.WindowBg)
	style.SetColor(imgui.StyleColorTitleBg, cols.TitleBg)
	style.SetColor(imgui.StyleColorTitleBgActive, cols.TitleBgActive)
	style.SetColor(imgui.StyleColorBorder, cols.Border)

	// convert 2600 colours to format usable by imgui
	cols.vec4PaletteNTSC = make(vec4Palette, 0, len(colors.PaletteNTSC))
	for _, c := range colors.PaletteNTSC {
		v := imgui.Vec4{
			float32(c.Red) / 255,
			float32(c.Green) / 255,
			float32(c.Blue) / 255,
			1.0,
		}
		cols.vec4PaletteNTSC = append(cols.vec4PaletteNTSC, v)
	}

	cols.vec4PalettePAL = make(vec4Palette, 0, len(colors.PalettePAL))
	for _, c := range colors.PalettePAL {
		v := imgui.Vec4{
			float32(c.Red) / 255,
			float32(c.Green) / 255,
			float32(c.Blue) / 255,
			1.0,
		}
		cols.vec4PalettePAL = append(cols.vec4PalettePAL, v)
	}

	cols.vec4PaletteAlt = make(vec4Palette, 0, len(colors.PaletteAlt))
	for _, c := range colors.PaletteAlt {
		v := imgui.Vec4{
			float32(c.Red) / 255,
			float32(c.Green) / 255,
			float32(c.Blue) / 255,
			1.0,
		}
		cols.vec4PaletteAlt = append(cols.vec4PaletteAlt, v)
	}

	cols.packedPaletteNTSC = make(packedPalette, 0, len(cols.vec4PaletteNTSC))
	for _, c := range cols.vec4PaletteNTSC {
		cols.packedPaletteNTSC = append(cols.packedPaletteNTSC, imgui.PackedColorFromVec4(c))
	}

	cols.packedPalettePAL = make(packedPalette, 0, len(cols.vec4PalettePAL))
	for _, c := range cols.vec4PalettePAL {
		cols.packedPalettePAL = append(cols.packedPalettePAL, imgui.PackedColorFromVec4(c))
	}

	cols.packedPaletteAlt = make(packedPalette, 0, len(cols.vec4PaletteAlt))
	for _, c := range cols.vec4PaletteAlt {
		cols.packedPaletteAlt = append(cols.packedPaletteAlt, imgui.PackedColorFromVec4(c))
	}

	return &cols
}
