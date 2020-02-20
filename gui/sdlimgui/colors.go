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
	"github.com/inkyblackness/imgui-go/v2"
)

// Colors defines all the colors used by the GUI
type Colors struct {
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
	CPUStatusOn         imgui.Vec4
	CPUStatusOnHovered  imgui.Vec4
	CPUStatusOnActive   imgui.Vec4
	CPUStatusOff        imgui.Vec4
	CPUStatusOffHovered imgui.Vec4
	CPUStatusOffActive  imgui.Vec4
	CPURdyFlagOn        imgui.Vec4
	CPURdyFlagOff       imgui.Vec4

	// control window buttons
	ControlRun         imgui.Vec4
	ControlRunHovered  imgui.Vec4
	ControlRunActive   imgui.Vec4
	ControlHalt        imgui.Vec4
	ControlHaltHovered imgui.Vec4
	ControlHaltActive  imgui.Vec4

	// disassembly entry columns
	DisasmCurrentPC   imgui.Vec4
	DisasmAddress     imgui.Vec4
	DisasmMnemonic    imgui.Vec4
	DisasmOperand     imgui.Vec4
	DisasmCycles      imgui.Vec4
	DisasmNotes       imgui.Vec4
	DisasmSelectedAdj imgui.Vec4

	// oscilloscope
	OscBg   imgui.Vec4
	OscLine imgui.Vec4

	// terminal
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

	// badges
	BreakpointPC imgui.Vec4
}

func defaultTheme() *Colors {
	cols := Colors{
		MenuBarBg:     imgui.Vec4{0.075, 0.08, 0.09, 1.0},
		WindowBg:      imgui.Vec4{0.075, 0.08, 0.09, 0.8},
		TitleBg:       imgui.Vec4{0.075, 0.08, 0.09, 1.0},
		TitleBgActive: imgui.Vec4{0.16, 0.29, 0.48, 1.0},
		Border:        imgui.Vec4{0.14, 0.14, 0.29, 1.0},

		// setting CapturedScreen* colors later

		CPUStatusOn:              imgui.Vec4{0.73, 0.49, 0.14, 1.0},
		CPUStatusOnHovered:       imgui.Vec4{0.79, 0.54, 0.15, 1.0},
		CPUStatusOnActive:        imgui.Vec4{0.79, 0.54, 0.15, 1.0},
		CPUStatusOff:             imgui.Vec4{0.64, 0.40, 0.09, 1.0},
		CPUStatusOffHovered:      imgui.Vec4{0.70, 0.45, 0.10, 1.0},
		CPUStatusOffActive:       imgui.Vec4{0.70, 0.45, 0.10, 1.0},
		CPURdyFlagOn:             imgui.Vec4{0.3, 0.6, 0.3, 1.0},
		CPURdyFlagOff:            imgui.Vec4{0.6, 0.3, 0.3, 1.0},
		ControlRun:               imgui.Vec4{0.3, 0.6, 0.3, 1.0},
		ControlRunHovered:        imgui.Vec4{0.3, 0.65, 0.3, 1.0},
		ControlRunActive:         imgui.Vec4{0.3, 0.65, 0.3, 1.0},
		ControlHalt:              imgui.Vec4{0.6, 0.3, 0.3, 1.0},
		ControlHaltHovered:       imgui.Vec4{0.65, 0.3, 0.3, 1.0},
		ControlHaltActive:        imgui.Vec4{0.65, 0.3, 0.3, 1.0},
		DisasmCurrentPC:          imgui.Vec4{0.9, 0.9, 0.9, 1.0},
		DisasmAddress:            imgui.Vec4{0.8, 0.4, 0.4, 1.0},
		DisasmMnemonic:           imgui.Vec4{0.4, 0.4, 0.8, 1.0},
		DisasmOperand:            imgui.Vec4{0.8, 0.8, 0.3, 1.0},
		DisasmCycles:             imgui.Vec4{0.8, 0.8, 0.8, 1.0},
		DisasmNotes:              imgui.Vec4{0.8, 0.8, 0.8, 1.0},
		DisasmSelectedAdj:        imgui.Vec4{0.1, 0.1, 0.1, 0.0},
		OscBg:                    imgui.Vec4{0.21, 0.29, 0.23, 1.0},
		OscLine:                  imgui.Vec4{0.10, 0.97, 0.29, 1.0},
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
		BreakpointPC:             imgui.Vec4{0.5, 1.0, 1.0, 0.5},
	}

	// set SapturedScreen* color to match default colors
	cols.CapturedScreenTitle = cols.TitleBgActive
	cols.CapturedScreenBorder = cols.TitleBgActive

	// set default colors
	style := imgui.CurrentStyle()
	style.SetColor(imgui.StyleColorMenuBarBg, cols.MenuBarBg)
	style.SetColor(imgui.StyleColorWindowBg, cols.WindowBg)
	style.SetColor(imgui.StyleColorTitleBg, cols.TitleBg)
	style.SetColor(imgui.StyleColorTitleBgActive, cols.TitleBgActive)
	style.SetColor(imgui.StyleColorBorder, cols.Border)

	return &cols
}
