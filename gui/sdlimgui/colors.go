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
	"image/color"

	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/hardware/tia/video"
	"github.com/jetsetilly/gopher2600/reflection"

	"github.com/inkyblackness/imgui-go/v4"
)

// packedPalette is an array of imgui.PackedColor.
type packedPalette []imgui.PackedColor

// imguiColors defines all the colors used by the GUI. Fields with leading
// uppercase are all of type imgui.Vec4. equivalent color but of tpye
// imgui.PackedColor have names with leading lowercase.
type imguiColors struct {
	// default colors
	MenuBarBg     imgui.Vec4
	WindowBg      imgui.Vec4
	TitleBg       imgui.Vec4
	TitleBgActive imgui.Vec4
	Border        imgui.Vec4

	// additional general colors
	True        imgui.Vec4
	False       imgui.Vec4
	TrueFalse   imgui.Vec4
	Transparent imgui.Vec4
	Warning     imgui.Vec4
	Cancel      imgui.Vec4

	// playscreen color
	PlayWindowBg     imgui.Vec4
	PlayWindowBorder imgui.Vec4

	// ROM selector
	ROMSelectDir  imgui.Vec4
	ROMSelectFile imgui.Vec4

	// the color to draw the TV Screen window border when mouse is captured
	CapturedScreenTitle  imgui.Vec4
	CapturedScreenBorder imgui.Vec4

	// value colors
	ValueDiff   imgui.Vec4
	ValueSymbol imgui.Vec4
	ValueStack  imgui.Vec4

	// control window buttons
	ControlRun         imgui.Vec4
	ControlRunHovered  imgui.Vec4
	ControlRunActive   imgui.Vec4
	ControlHalt        imgui.Vec4
	ControlHaltHovered imgui.Vec4
	ControlHaltActive  imgui.Vec4

	// cpu
	CPURDY    imgui.Vec4
	CPUNotRDY imgui.Vec4
	CPUKIL    imgui.Vec4

	// disassembly entry columns
	DisasmLocation imgui.Vec4
	DisasmBank     imgui.Vec4
	DisasmAddress  imgui.Vec4
	DisasmByteCode imgui.Vec4
	DisasmOperator imgui.Vec4
	DisasmOperand  imgui.Vec4
	DisasmCycles   imgui.Vec4
	DisasmNotes    imgui.Vec4

	// disassembly other
	DisasmStep         imgui.Vec4
	DisasmHover        imgui.Vec4
	DisasmBreakAddress imgui.Vec4

	// coprocessor source (and related) windows
	CoProcSourceSelected         imgui.Vec4
	CoProcSourceYield            imgui.Vec4
	CoProcSourceYieldBug         imgui.Vec4
	CoProcSourceHover            imgui.Vec4
	CoProcSourceFilename         imgui.Vec4
	CoProcSourceLineNumber       imgui.Vec4
	CoProcSourceLoad             imgui.Vec4
	CoProcSourceAvgLoad          imgui.Vec4
	CoProcSourceMaxLoad          imgui.Vec4
	CoProcSourceNoLoad           imgui.Vec4
	CoProcSourceBug              imgui.Vec4
	CoProcSourceChip             imgui.Vec4
	CoProcSourceDisasmOpcode     imgui.Vec4
	CoProcSourceDisasmOpcodeFade imgui.Vec4
	CoProcSourceDisasmAddr       imgui.Vec4
	CoProcSourceDisasmAddrFade   imgui.Vec4
	CoProcSourceDisasm           imgui.Vec4
	CoProcSourceDisasmFade       imgui.Vec4
	CoProcSourceComment          imgui.Vec4
	CoProcSourceStringLiteral    imgui.Vec4
	CoProcIllegalAccessAddress   imgui.Vec4
	CoProcIllegalAccessFrequency imgui.Vec4
	CoProcIllegalAccessNotes     imgui.Vec4
	CoProcVariablesType          imgui.Vec4
	CoProcVariablesTypeSize      imgui.Vec4
	CoProcVariablesAddress       imgui.Vec4
	CoProcVariablesNotes         imgui.Vec4

	// coprocessor last execution
	CoProcMAM0               imgui.Vec4
	CoProcMAM1               imgui.Vec4
	CoProcMAM2               imgui.Vec4
	CoProcBranchTrailUsed    imgui.Vec4
	CoProcBranchTrailFlushed imgui.Vec4
	CoProcMergedIS           imgui.Vec4

	// datastream window
	DataStreamNumLabel imgui.Vec4

	// audio oscilloscope
	AudioOscBg   imgui.Vec4
	AudioOscLine imgui.Vec4

	// audio tracker
	AudioTrackerHeader         imgui.Vec4
	AudioTrackerRow            imgui.Vec4
	AudioTrackerRowAlt         imgui.Vec4
	AudioTrackerRowSelected    imgui.Vec4
	AudioTrackerRowSelectedAlt imgui.Vec4
	AudioTrackerRowHover       imgui.Vec4
	AudioTrackerBorder         imgui.Vec4

	// piano keys
	PianoKeysBackground imgui.Vec4
	PianoKeysBorder     imgui.Vec4

	// timeline plot
	TimelineMarkers        imgui.Vec4
	TimelineScanlines      imgui.Vec4
	TimelineWSYNC          imgui.Vec4
	TimelineCoProc         imgui.Vec4
	TimelineRewindRange    imgui.Vec4
	TimelineCurrentPointer imgui.Vec4
	TimelineCmpPointer     imgui.Vec4
	TimelineLeftPlayer     imgui.Vec4

	// tia window
	TIApointer imgui.Vec4

	// collision window
	CollisionBit imgui.Vec4

	// ports window
	PortsBit imgui.Vec4

	// timer window
	TimerBit imgui.Vec4

	// savekey i2c/eeprom window
	SaveKeyBit        imgui.Vec4
	SaveKeyOscBG      imgui.Vec4
	SaveKeyOscSCL     imgui.Vec4
	SaveKeyOscSDA     imgui.Vec4
	SaveKeyBitPointer imgui.Vec4

	// terminal
	TermBackground      imgui.Vec4
	TermInput           imgui.Vec4
	TermStyleEcho       imgui.Vec4
	TermStyleHelp       imgui.Vec4
	TermStyleFeedback   imgui.Vec4
	TermStyleCPUStep    imgui.Vec4
	TermStyleVideoStep  imgui.Vec4
	TermStyleInstrument imgui.Vec4
	TermStyleError      imgui.Vec4
	TermStyleLog        imgui.Vec4

	// helpers
	ToolTipBG imgui.Vec4

	// log
	LogBackground        imgui.Vec4
	LogMultilineEmphasis imgui.Vec4

	// packed equivalents of the above colors (where appropriate)
	windowBg               imgui.PackedColor
	tiaPointer             imgui.PackedColor
	collisionBit           imgui.PackedColor
	portsBit               imgui.PackedColor
	timerBit               imgui.PackedColor
	saveKeyBit             imgui.PackedColor
	saveKeyBitPointer      imgui.PackedColor
	timelineMarkers        imgui.PackedColor
	timelineScanlines      imgui.PackedColor
	timelineWSYNC          imgui.PackedColor
	timelineCoProc         imgui.PackedColor
	timelineRewindRange    imgui.PackedColor
	timelineCurrentPointer imgui.PackedColor
	timelineCmpPointer     imgui.PackedColor
	timelineLeftPlayer     imgui.PackedColor
	coProcSourceLoad       imgui.PackedColor
	coProcSourceAvgLoad    imgui.PackedColor
	coProcSourceMaxLoad    imgui.PackedColor
	coProcSourceNoLoad     imgui.PackedColor

	// reflection colors
	reflectionColors []imgui.Vec4

	// packed TV palettes
	packedPaletteNTSC packedPalette
	packedPalettePAL  packedPalette
	paletteNTSC       []imgui.Vec4
	palettePAL        []imgui.Vec4
}

func newColors() *imguiColors {
	cols := imguiColors{
		// default colors
		MenuBarBg:     imgui.Vec4{0.075, 0.08, 0.09, 1.0},
		WindowBg:      imgui.Vec4{0.075, 0.08, 0.09, 1.0},
		TitleBg:       imgui.Vec4{0.075, 0.08, 0.09, 1.0},
		TitleBgActive: imgui.Vec4{0.16, 0.29, 0.48, 1.0},
		Border:        imgui.Vec4{0.14, 0.14, 0.29, 1.0},

		// additional general colors
		True:        imgui.Vec4{0.3, 0.6, 0.3, 1.0},
		False:       imgui.Vec4{0.6, 0.3, 0.3, 1.0},
		TrueFalse:   imgui.Vec4{0.6, 0.6, 0.3, 1.0},
		Transparent: imgui.Vec4{0.0, 0.0, 0.0, 0.0},
		Warning:     imgui.Vec4{1.0, 0.2, 0.2, 1.0},
		Cancel:      imgui.Vec4{1.0, 0.2, 0.2, 1.0},

		// playscreen color
		PlayWindowBg:     imgui.Vec4{0.0, 0.0, 0.0, 1.0},
		PlayWindowBorder: imgui.Vec4{0.0, 0.0, 0.0, 1.0},

		// ROM selector
		ROMSelectDir:  imgui.Vec4{1.0, 0.5, 0.5, 1.0},
		ROMSelectFile: imgui.Vec4{1.0, 1.0, 1.0, 1.0},

		// deferring CapturedScreenTitle & CapturedScreenBorder

		// value colors
		ValueDiff:   imgui.Vec4{0.3, 0.2, 0.4, 1.0},
		ValueSymbol: imgui.Vec4{0.6, 0.6, 0.0, 1.0},
		ValueStack:  imgui.Vec4{0.5, 0.2, 0.3, 1.0},

		// control window buttons
		ControlRun:         imgui.Vec4{0.3, 0.6, 0.3, 1.0},
		ControlRunHovered:  imgui.Vec4{0.3, 0.65, 0.3, 1.0},
		ControlRunActive:   imgui.Vec4{0.3, 0.65, 0.3, 1.0},
		ControlHalt:        imgui.Vec4{0.6, 0.3, 0.3, 1.0},
		ControlHaltHovered: imgui.Vec4{0.65, 0.3, 0.3, 1.0},
		ControlHaltActive:  imgui.Vec4{0.65, 0.3, 0.3, 1.0},

		// cpu window
		CPURDY:    imgui.Vec4{0.3, 0.6, 0.3, 1.0},
		CPUNotRDY: imgui.Vec4{0.6, 0.3, 0.3, 1.0},
		CPUKIL:    imgui.Vec4{0.2, 0.2, 0.2, 1.0},

		// disassembly entry columns
		DisasmLocation: imgui.Vec4{0.8, 0.8, 0.8, 1.0},
		DisasmBank:     imgui.Vec4{0.6, 0.6, 0.8, 1.0},
		DisasmAddress:  imgui.Vec4{0.8, 0.4, 0.4, 1.0},
		DisasmByteCode: imgui.Vec4{0.5, 0.5, 0.6, 1.0},
		DisasmOperator: imgui.Vec4{0.4, 0.4, 0.8, 1.0},
		DisasmOperand:  imgui.Vec4{0.8, 0.8, 0.3, 1.0},
		DisasmCycles:   imgui.Vec4{0.8, 0.8, 0.8, 1.0},
		DisasmNotes:    imgui.Vec4{0.8, 0.8, 0.8, 1.0},

		// disassembly other
		DisasmStep:         imgui.Vec4{1.0, 1.0, 1.0, 0.1},
		DisasmHover:        imgui.Vec4{0.5, 0.5, 0.5, 0.1},
		DisasmBreakAddress: imgui.Vec4{0.9, 0.4, 0.4, 1.0},

		// coprocessor source (and related) windows
		CoProcSourceSelected:         imgui.Vec4{1.0, 1.0, 1.0, 0.1},
		CoProcSourceYield:            imgui.Vec4{0.5, 1.0, 0.5, 0.1},
		CoProcSourceYieldBug:         imgui.Vec4{1.0, 0.5, 0.5, 0.1},
		CoProcSourceHover:            imgui.Vec4{0.5, 0.5, 0.5, 0.1},
		CoProcSourceFilename:         imgui.Vec4{0.7, 0.7, 0.8, 1.0},
		CoProcSourceLineNumber:       imgui.Vec4{0.5, 0.5, 0.8, 1.0},
		CoProcSourceLoad:             imgui.Vec4{0.8, 0.5, 0.5, 1.0},
		CoProcSourceAvgLoad:          imgui.Vec4{0.5, 0.5, 0.8, 1.0},
		CoProcSourceMaxLoad:          imgui.Vec4{0.8, 0.5, 0.7, 1.0},
		CoProcSourceNoLoad:           imgui.Vec4{0.5, 0.5, 0.5, 1.0},
		CoProcSourceBug:              imgui.Vec4{0.5, 0.5, 0.8, 1.0},
		CoProcSourceChip:             imgui.Vec4{0.5, 0.5, 0.5, 1.0},
		CoProcSourceDisasmOpcode:     imgui.Vec4{0.5, 0.5, 0.6, 1.0},
		CoProcSourceDisasmOpcodeFade: imgui.Vec4{0.3, 0.3, 0.3, 1.0},
		CoProcSourceDisasmAddr:       imgui.Vec4{0.8, 0.8, 0.8, 1.0},
		CoProcSourceDisasmAddrFade:   imgui.Vec4{0.5, 0.5, 0.5, 1.0},
		CoProcSourceDisasm:           imgui.Vec4{1.0, 1.0, 1.0, 1.0},
		CoProcSourceDisasmFade:       imgui.Vec4{0.6, 0.6, 0.6, 1.0},
		CoProcSourceComment:          imgui.Vec4{0.4, 0.4, 0.6, 1.0},
		CoProcSourceStringLiteral:    imgui.Vec4{0.4, 0.6, 0.6, 1.0},
		CoProcIllegalAccessAddress:   imgui.Vec4{0.8, 0.4, 0.4, 1.0},
		CoProcIllegalAccessFrequency: imgui.Vec4{0.7, 0.7, 0.7, 1.0},
		CoProcIllegalAccessNotes:     imgui.Vec4{0.7, 0.7, 0.7, 1.0},
		CoProcVariablesType:          imgui.Vec4{0.8, 0.6, 0.8, 1.0},
		CoProcVariablesTypeSize:      imgui.Vec4{0.8, 0.6, 0.6, 1.0},
		CoProcVariablesAddress:       imgui.Vec4{0.8, 0.4, 0.4, 1.0},
		CoProcVariablesNotes:         imgui.Vec4{0.8, 0.8, 0.8, 1.0},

		// coprocessor disassembly
		CoProcMAM0:               imgui.Vec4{0.6, 0.3, 0.3, 1.0},
		CoProcMAM1:               imgui.Vec4{0.6, 0.6, 0.3, 1.0},
		CoProcMAM2:               imgui.Vec4{0.3, 0.6, 0.3, 1.0},
		CoProcBranchTrailFlushed: imgui.Vec4{0.6, 0.3, 0.3, 1.0},
		CoProcBranchTrailUsed:    imgui.Vec4{0.3, 0.6, 0.3, 1.0},
		CoProcMergedIS:           imgui.Vec4{0.3, 0.3, 0.6, 1.0},

		// datastream window
		DataStreamNumLabel: imgui.Vec4{0.7, 0.7, 0.7, 1.0},

		// audio oscilloscope
		AudioOscBg:   imgui.Vec4{0.21, 0.29, 0.23, 1.0},
		AudioOscLine: imgui.Vec4{0.10, 0.97, 0.29, 1.0},

		// audio tracker
		AudioTrackerHeader:         imgui.Vec4{0.12, 0.25, 0.25, 1.0},
		AudioTrackerRow:            imgui.Vec4{0.10, 0.15, 0.15, 1.0},
		AudioTrackerRowAlt:         imgui.Vec4{0.12, 0.17, 0.17, 1.0},
		AudioTrackerRowSelected:    imgui.Vec4{0.15, 0.25, 0.25, 1.0},
		AudioTrackerRowSelectedAlt: imgui.Vec4{0.17, 0.27, 0.27, 1.0},
		AudioTrackerRowHover:       imgui.Vec4{0.14, 0.20, 0.20, 1.0},
		AudioTrackerBorder:         imgui.Vec4{0.12, 0.25, 0.25, 1.0},

		// piano keys
		PianoKeysBackground: imgui.Vec4{0.10, 0.09, 0.05, 1.0},
		PianoKeysBorder:     imgui.Vec4{0.29, 0.20, 0.14, 1.0},

		// timeline
		TimelineMarkers:   imgui.Vec4{1.00, 1.00, 1.00, 1.0},
		TimelineScanlines: imgui.Vec4{0.79, 0.04, 0.04, 1.0},
		// deferred TimelineWSYNC and TimelineCoProc
		TimelineRewindRange:    imgui.Vec4{0.79, 0.38, 0.04, 1.0},
		TimelineCurrentPointer: imgui.Vec4{0.79, 0.38, 0.04, 1.0}, // same as TimelineRewindRange
		TimelineCmpPointer:     imgui.Vec4{0.30, 0.20, 0.50, 1.0}, // same as ValueDiff
		TimelineLeftPlayer:     imgui.Vec4{0.38, 0.79, 0.04, 1.0}, // same as TimelineRewindRange

		// tia
		TIApointer: imgui.Vec4{0.8, 0.8, 0.8, 1.0},

		// deffering collision window CollisionBit

		// deferring chip registers window RegisterBit

		// deferring savekey i2c/eeprom window RegisterBit

		SaveKeyOscBG:      imgui.Vec4{0.10, 0.10, 0.10, 1.0},
		SaveKeyOscSCL:     imgui.Vec4{0.10, 0.97, 0.29, 1.0},
		SaveKeyOscSDA:     imgui.Vec4{0.97, 0.10, 0.29, 1.0},
		SaveKeyBitPointer: imgui.Vec4{0.8, 0.8, 0.8, 1.0},

		// terminal
		TermBackground:      imgui.Vec4{0.1, 0.1, 0.2, 0.9},
		TermInput:           imgui.Vec4{0.1, 0.1, 0.25, 0.9},
		TermStyleEcho:       imgui.Vec4{0.8, 0.8, 0.8, 1.0},
		TermStyleHelp:       imgui.Vec4{1.0, 1.0, 1.0, 1.0},
		TermStyleFeedback:   imgui.Vec4{1.0, 1.0, 1.0, 1.0},
		TermStyleCPUStep:    imgui.Vec4{0.9, 0.9, 0.5, 1.0},
		TermStyleVideoStep:  imgui.Vec4{0.7, 0.7, 0.3, 1.0},
		TermStyleInstrument: imgui.Vec4{0.1, 0.95, 0.9, 1.0},
		TermStyleError:      imgui.Vec4{0.8, 0.3, 0.3, 1.0},
		TermStyleLog:        imgui.Vec4{0.8, 0.7, 0.3, 1.0},

		// helpers
		ToolTipBG: imgui.Vec4{0.2, 0.1, 0.2, 0.8},

		// log
		LogBackground:        imgui.Vec4{0.2, 0.2, 0.3, 0.9},
		LogMultilineEmphasis: imgui.Vec4{1.0, 0.5, 0.5, 1.0},
	}

	// set default colors
	style := imgui.CurrentStyle()
	style.SetColor(imgui.StyleColorMenuBarBg, cols.MenuBarBg)
	style.SetColor(imgui.StyleColorWindowBg, cols.WindowBg)
	style.SetColor(imgui.StyleColorTitleBg, cols.TitleBg)
	style.SetColor(imgui.StyleColorTitleBgActive, cols.TitleBgActive)
	style.SetColor(imgui.StyleColorBorder, cols.Border)

	// reflection colors in imgui.Vec4 and imgui.PackedColor formats
	cols.reflectionColors = make([]imgui.Vec4, len(reflectionColors))
	for i, v := range reflectionColors {
		c := imgui.Vec4{float32(v.R) / 255.0, float32(v.G) / 255.0, float32(v.B) / 255.0, float32(v.A) / 255.0}
		cols.reflectionColors[i] = c
	}

	// we deferred setting of some colours. set them now.
	cols.CapturedScreenTitle = cols.TitleBgActive
	cols.CapturedScreenBorder = cols.TitleBgActive
	cols.CollisionBit = imgui.CurrentStyle().Color(imgui.StyleColorButton)
	cols.PortsBit = imgui.CurrentStyle().Color(imgui.StyleColorButton)
	cols.TimerBit = imgui.CurrentStyle().Color(imgui.StyleColorButton)
	cols.SaveKeyBit = imgui.CurrentStyle().Color(imgui.StyleColorButton)
	cols.TimelineWSYNC = cols.reflectionColors[reflection.WSYNC]
	cols.TimelineCoProc = cols.reflectionColors[reflection.CoProcActive]

	// colors that are used in context where an imgui.PackedColor is required
	cols.windowBg = imgui.PackedColorFromVec4(cols.WindowBg)
	cols.tiaPointer = imgui.PackedColorFromVec4(cols.TIApointer)
	cols.collisionBit = imgui.PackedColorFromVec4(cols.CollisionBit)
	cols.portsBit = imgui.PackedColorFromVec4(cols.PortsBit)
	cols.timerBit = imgui.PackedColorFromVec4(cols.TimerBit)
	cols.saveKeyBit = imgui.PackedColorFromVec4(cols.SaveKeyBit)
	cols.saveKeyBitPointer = imgui.PackedColorFromVec4(cols.SaveKeyBitPointer)
	cols.timelineMarkers = imgui.PackedColorFromVec4(cols.TimelineMarkers)
	cols.timelineScanlines = imgui.PackedColorFromVec4(cols.TimelineScanlines)
	cols.timelineWSYNC = imgui.PackedColorFromVec4(cols.TimelineWSYNC)
	cols.timelineCoProc = imgui.PackedColorFromVec4(cols.TimelineCoProc)
	cols.timelineRewindRange = imgui.PackedColorFromVec4(cols.TimelineRewindRange)
	cols.timelineCurrentPointer = imgui.PackedColorFromVec4(cols.TimelineCurrentPointer)
	cols.timelineCmpPointer = imgui.PackedColorFromVec4(cols.TimelineCmpPointer)
	cols.timelineLeftPlayer = imgui.PackedColorFromVec4(cols.TimelineLeftPlayer)
	cols.coProcSourceLoad = imgui.PackedColorFromVec4(cols.CoProcSourceLoad)
	cols.coProcSourceAvgLoad = imgui.PackedColorFromVec4(cols.CoProcSourceAvgLoad)
	cols.coProcSourceMaxLoad = imgui.PackedColorFromVec4(cols.CoProcSourceMaxLoad)
	cols.coProcSourceNoLoad = imgui.PackedColorFromVec4(cols.CoProcSourceNoLoad)

	// convert 2600 colours to format usable by imgui

	// convert to imgui.Vec4 first...
	cols.paletteNTSC = make([]imgui.Vec4, 0, len(specification.PaletteNTSC))
	for _, c := range specification.PaletteNTSC {
		v := imgui.Vec4{
			float32(c.R) / 255,
			float32(c.G) / 255,
			float32(c.B) / 255,
			1.0,
		}
		cols.paletteNTSC = append(cols.paletteNTSC, v)
	}

	cols.palettePAL = make([]imgui.Vec4, 0, len(specification.PalettePAL))
	for _, c := range specification.PalettePAL {
		v := imgui.Vec4{
			float32(c.R) / 255,
			float32(c.G) / 255,
			float32(c.B) / 255,
			1.0,
		}
		cols.palettePAL = append(cols.palettePAL, v)
	}

	// ...then to the packedPalette
	cols.packedPaletteNTSC = make(packedPalette, 0, len(cols.paletteNTSC))
	for _, c := range cols.paletteNTSC {
		cols.packedPaletteNTSC = append(cols.packedPaletteNTSC, imgui.PackedColorFromVec4(c))
	}

	cols.packedPalettePAL = make(packedPalette, 0, len(cols.packedPalettePAL))
	for _, c := range cols.palettePAL {
		cols.packedPalettePAL = append(cols.packedPalettePAL, imgui.PackedColorFromVec4(c))
	}

	return &cols
}

// reflectionColors lists the colors to be used for the reflection overlay.
var reflectionColors = []color.RGBA{
	reflection.WSYNC:        {R: 50, G: 50, B: 255, A: 255},
	reflection.Collision:    {R: 255, G: 25, B: 25, A: 255},
	reflection.CXCLR:        {R: 255, G: 25, B: 255, A: 255},
	reflection.HMOVEdelay:   {R: 150, G: 50, B: 50, A: 255},
	reflection.HMOVEripple:  {R: 50, G: 150, B: 50, A: 255},
	reflection.HMOVElatched: {R: 50, G: 50, B: 150, A: 255},
	reflection.RSYNCalign:   {R: 50, G: 50, B: 200, A: 255},
	reflection.RSYNCreset:   {R: 50, G: 200, B: 200, A: 255},
	reflection.AudioPhase0:  {R: 250, G: 100, B: 100, A: 255},
	reflection.AudioPhase1:  {R: 100, G: 250, B: 100, A: 255},
	reflection.AudioChanged: {R: 150, G: 150, B: 250, A: 255},

	reflection.CoProcInactive: {R: 0, G: 0, B: 0, A: 0},
	reflection.CoProcActive:   {R: 200, G: 50, B: 200, A: 255},
}

// altColors lists the colors to be used when displaying TIA video in a
// debugger's "debug colors" mode. these colors are the same as the the debug
// colors found in the Stella emulator.
var altColors = []color.RGBA{
	video.ElementBackground: {R: 17, G: 17, B: 17, A: 255},
	video.ElementBall:       {R: 132, G: 200, B: 252, A: 255},
	video.ElementPlayfield:  {R: 146, G: 70, B: 192, A: 255},
	video.ElementPlayer0:    {R: 144, G: 28, B: 0, A: 255},
	video.ElementPlayer1:    {R: 232, G: 232, B: 74, A: 255},
	video.ElementMissile0:   {R: 213, G: 130, B: 74, A: 255},
	video.ElementMissile1:   {R: 50, G: 132, B: 50, A: 255},
}
