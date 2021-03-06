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
	Transparent imgui.Vec4

	// playscreen color
	PlayWindowBg     imgui.Vec4
	PlayWindowBorder imgui.Vec4

	// ROM selector
	ROMSelectDir  imgui.Vec4
	ROMSelectFile imgui.Vec4

	// the color to draw the TV Screen window border when mouse is captured
	CapturedScreenTitle  imgui.Vec4
	CapturedScreenBorder imgui.Vec4

	// color showing that a value is different to the corresponding value at
	// the comparison point
	ValueDiff imgui.Vec4

	// control window buttons
	ControlRun         imgui.Vec4
	ControlRunHovered  imgui.Vec4
	ControlRunActive   imgui.Vec4
	ControlHalt        imgui.Vec4
	ControlHaltHovered imgui.Vec4
	ControlHaltActive  imgui.Vec4

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
	DisasmCPUstep      imgui.Vec4
	DisasmVideoStep    imgui.Vec4
	DisasmBreakAddress imgui.Vec4
	DisasmBreakOther   imgui.Vec4

	// audio oscilloscope
	AudioOscBg   imgui.Vec4
	AudioOscLine imgui.Vec4

	// tia window
	TIApointer imgui.Vec4

	// collision window
	CollisionBit imgui.Vec4

	// chip registers window
	RegisterBit imgui.Vec4

	// savekey i2c/eeprom window
	SaveKeyBit        imgui.Vec4
	SaveKeyOscBG      imgui.Vec4
	SaveKeyOscSCL     imgui.Vec4
	SaveKeyOscSDA     imgui.Vec4
	SaveKeyBitPointer imgui.Vec4

	// terminal
	TermBackground      imgui.Vec4
	TermStyleEcho       imgui.Vec4
	TermStyleHelp       imgui.Vec4
	TermStyleFeedback   imgui.Vec4
	TermStyleCPUStep    imgui.Vec4
	TermStyleVideoStep  imgui.Vec4
	TermStyleInstrument imgui.Vec4
	TermStyleError      imgui.Vec4
	TermStyleLog        imgui.Vec4

	// log
	LogBackground imgui.Vec4

	// packed equivalents of the above colors (where appropriate)
	disasmCPUstep      imgui.PackedColor
	disasmVideoStep    imgui.PackedColor
	disasmBreakAddress imgui.PackedColor
	disasmBreakOther   imgui.PackedColor
	tiaPointer         imgui.PackedColor
	collisionBit       imgui.PackedColor
	registerBit        imgui.PackedColor
	saveKeyBit         imgui.PackedColor
	saveKeyOscSCL      imgui.PackedColor
	saveKeyOscSDA      imgui.PackedColor
	saveKeyBitPointer  imgui.PackedColor
	trueCol            imgui.PackedColor
	falseCol           imgui.PackedColor

	// packed reflection colors
	reflectionColors map[reflection.ID]imgui.PackedColor

	// packed TV palettes
	packedPaletteNTSC packedPalette
	packedPalettePAL  packedPalette
}

func newColors() *imguiColors {
	cols := imguiColors{
		// default colors
		MenuBarBg:     imgui.Vec4{0.075, 0.08, 0.09, 1.0},
		WindowBg:      imgui.Vec4{0.075, 0.08, 0.09, 1.0},
		TitleBg:       imgui.Vec4{0.075, 0.08, 0.09, 1.0},
		TitleBgActive: imgui.Vec4{0.16, 0.29, 0.48, 1.0},
		Border:        imgui.Vec4{0.14, 0.14, 0.29, 1.0},

		PlayWindowBg:     imgui.Vec4{0.0, 0.0, 0.0, 1.0},
		PlayWindowBorder: imgui.Vec4{0.0, 0.0, 0.0, 1.0},

		// additional general colors
		True:        imgui.Vec4{0.3, 0.6, 0.3, 1.0},
		False:       imgui.Vec4{0.6, 0.3, 0.3, 1.0},
		Transparent: imgui.Vec4{0.0, 0.0, 0.0, 0.0},

		// ROM selector
		ROMSelectDir:  imgui.Vec4{1.0, 0.5, 0.5, 1.0},
		ROMSelectFile: imgui.Vec4{1.0, 1.0, 1.0, 1.0},

		// deferring CapturedScreenTitle & CapturedScreenBorder

		// comparison
		ValueDiff: imgui.Vec4{0.3, 0.2, 0.5, 1.0},

		// control window buttons
		ControlRun:         imgui.Vec4{0.3, 0.6, 0.3, 1.0},
		ControlRunHovered:  imgui.Vec4{0.3, 0.65, 0.3, 1.0},
		ControlRunActive:   imgui.Vec4{0.3, 0.65, 0.3, 1.0},
		ControlHalt:        imgui.Vec4{0.6, 0.3, 0.3, 1.0},
		ControlHaltHovered: imgui.Vec4{0.65, 0.3, 0.3, 1.0},
		ControlHaltActive:  imgui.Vec4{0.65, 0.3, 0.3, 1.0},

		// disassembly entry columns
		DisasmLocation: imgui.Vec4{0.8, 0.8, 0.8, 1.0},
		DisasmBank:     imgui.Vec4{0.8, 0.8, 0.8, 1.0},
		DisasmAddress:  imgui.Vec4{0.8, 0.4, 0.4, 1.0},
		DisasmByteCode: imgui.Vec4{0.6, 0.3, 0.4, 1.0},
		DisasmOperator: imgui.Vec4{0.4, 0.4, 0.8, 1.0},
		DisasmOperand:  imgui.Vec4{0.8, 0.8, 0.3, 1.0},
		DisasmCycles:   imgui.Vec4{0.8, 0.8, 0.8, 1.0},
		DisasmNotes:    imgui.Vec4{0.8, 0.8, 0.8, 1.0},

		// disassembly other
		DisasmCPUstep:   imgui.Vec4{1.0, 1.0, 1.0, 0.1},
		DisasmVideoStep: imgui.Vec4{0.5, 0.5, 0.5, 0.07},
		// deferring DisasmBreakAddress & DisasmBreakOther

		// audio oscilloscope
		AudioOscBg:   imgui.Vec4{0.21, 0.29, 0.23, 1.0},
		AudioOscLine: imgui.Vec4{0.10, 0.97, 0.29, 1.0},

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
		TermStyleEcho:       imgui.Vec4{0.8, 0.8, 0.8, 1.0},
		TermStyleHelp:       imgui.Vec4{1.0, 1.0, 1.0, 1.0},
		TermStyleFeedback:   imgui.Vec4{1.0, 1.0, 1.0, 1.0},
		TermStyleCPUStep:    imgui.Vec4{0.9, 0.9, 0.5, 1.0},
		TermStyleVideoStep:  imgui.Vec4{0.7, 0.7, 0.3, 1.0},
		TermStyleInstrument: imgui.Vec4{0.1, 0.95, 0.9, 1.0},
		TermStyleError:      imgui.Vec4{0.8, 0.3, 0.3, 1.0},
		TermStyleLog:        imgui.Vec4{0.8, 0.7, 0.3, 1.0},

		// log
		LogBackground: imgui.Vec4{0.2, 0.2, 0.3, 0.9},
	}

	// set default colors
	style := imgui.CurrentStyle()
	style.SetColor(imgui.StyleColorMenuBarBg, cols.MenuBarBg)
	style.SetColor(imgui.StyleColorWindowBg, cols.WindowBg)
	style.SetColor(imgui.StyleColorTitleBg, cols.TitleBg)
	style.SetColor(imgui.StyleColorTitleBgActive, cols.TitleBgActive)
	style.SetColor(imgui.StyleColorBorder, cols.Border)

	// we deferred setting of some colours. set them now.
	cols.CapturedScreenTitle = cols.TitleBgActive
	cols.CapturedScreenBorder = cols.TitleBgActive
	cols.DisasmBreakAddress = cols.DisasmAddress
	cols.DisasmBreakOther = cols.DisasmOperator
	cols.CollisionBit = imgui.CurrentStyle().Color(imgui.StyleColorButton)
	cols.RegisterBit = imgui.CurrentStyle().Color(imgui.StyleColorButton)
	cols.SaveKeyBit = imgui.CurrentStyle().Color(imgui.StyleColorButton)

	// colors that are used in context where an imgui.PackedColor is required
	cols.disasmCPUstep = imgui.PackedColorFromVec4(cols.DisasmCPUstep)
	cols.disasmVideoStep = imgui.PackedColorFromVec4(cols.DisasmVideoStep)
	cols.disasmBreakAddress = imgui.PackedColorFromVec4(cols.DisasmBreakAddress)
	cols.disasmBreakOther = imgui.PackedColorFromVec4(cols.DisasmBreakOther)
	cols.tiaPointer = imgui.PackedColorFromVec4(cols.TIApointer)
	cols.collisionBit = imgui.PackedColorFromVec4(cols.CollisionBit)
	cols.registerBit = imgui.PackedColorFromVec4(cols.RegisterBit)
	cols.saveKeyBit = imgui.PackedColorFromVec4(cols.SaveKeyBit)
	cols.saveKeyBitPointer = imgui.PackedColorFromVec4(cols.SaveKeyBitPointer)
	cols.saveKeyOscSCL = imgui.PackedColorFromVec4(cols.SaveKeyOscSCL)
	cols.saveKeyOscSDA = imgui.PackedColorFromVec4(cols.SaveKeyOscSDA)
	cols.trueCol = imgui.PackedColorFromVec4(cols.True)
	cols.falseCol = imgui.PackedColorFromVec4(cols.False)

	// pack reflection colors
	cols.reflectionColors = make(map[reflection.ID]imgui.PackedColor)
	for k, v := range reflectionColors {
		c := imgui.Vec4{float32(v.R) / 255.0, float32(v.G) / 255.0, float32(v.B) / 255.0, float32(v.A) / 255.0}
		cols.reflectionColors[k] = imgui.PackedColorFromVec4(c)
	}

	// convert 2600 colours to format usable by imgui

	// convert to imgiu.Vec4 first...
	vec4PaletteNTSC := make([]imgui.Vec4, 0, len(specification.PaletteNTSC))
	for _, c := range specification.PaletteNTSC {
		v := imgui.Vec4{
			float32(c.R) / 255,
			float32(c.G) / 255,
			float32(c.B) / 255,
			1.0,
		}
		vec4PaletteNTSC = append(vec4PaletteNTSC, v)
	}

	vec4PalettePAL := make([]imgui.Vec4, 0, len(specification.PalettePAL))
	for _, c := range specification.PalettePAL {
		v := imgui.Vec4{
			float32(c.R) / 255,
			float32(c.G) / 255,
			float32(c.B) / 255,
			1.0,
		}
		vec4PalettePAL = append(vec4PalettePAL, v)
	}

	// ...then to the packedPalette
	cols.packedPaletteNTSC = make(packedPalette, 0, len(vec4PaletteNTSC))
	for _, c := range vec4PaletteNTSC {
		cols.packedPaletteNTSC = append(cols.packedPaletteNTSC, imgui.PackedColorFromVec4(c))
	}

	cols.packedPalettePAL = make(packedPalette, 0, len(vec4PalettePAL))
	for _, c := range vec4PalettePAL {
		cols.packedPalettePAL = append(cols.packedPalettePAL, imgui.PackedColorFromVec4(c))
	}

	return &cols
}

// reflectionColors lists the colors to be used for the reflection overlay.
var reflectionColors = map[reflection.ID]color.RGBA{
	reflection.WSYNC:             {R: 50, G: 50, B: 255, A: 255},
	reflection.Collision:         {R: 255, G: 25, B: 25, A: 255},
	reflection.CXCLR:             {R: 255, G: 25, B: 255, A: 255},
	reflection.HMOVEdelay:        {R: 150, G: 50, B: 50, A: 255},
	reflection.HMOVEripple:       {R: 50, G: 150, B: 50, A: 255},
	reflection.HMOVElatched:      {R: 50, G: 50, B: 150, A: 255},
	reflection.RSYNCalign:        {R: 50, G: 50, B: 200, A: 255},
	reflection.RSYNCreset:        {R: 50, G: 200, B: 200, A: 255},
	reflection.CoprocessorActive: {R: 200, G: 50, B: 200, A: 255},
}

// altColors lists the colors to be used when displaying TIA video in a
// debugger's "debug colors" mode. these colors are the same as the the debug
// colors found in the Stella emulator.
var altColors = map[video.Element]color.RGBA{
	video.ElementBackground: {R: 17, G: 17, B: 17, A: 255},
	video.ElementBall:       {R: 132, G: 200, B: 252, A: 255},
	video.ElementPlayfield:  {R: 146, G: 70, B: 192, A: 255},
	video.ElementPlayer0:    {R: 144, G: 28, B: 0, A: 255},
	video.ElementPlayer1:    {R: 232, G: 232, B: 74, A: 255},
	video.ElementMissile0:   {R: 213, G: 130, B: 74, A: 255},
	video.ElementMissile1:   {R: 50, G: 132, B: 50, A: 255},
}
