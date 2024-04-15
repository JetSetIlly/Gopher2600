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

package peripherals

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/atarivox"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/controllers"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/savekey"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

// Fingerprint scans the raw cartridge data for patterns that indicate the
// requirement of a specific controller type.
//
// The patterns in this file are taken from the Stella project. Specifically
// the following file (last retreived on 28th January 2022).
//
// https://github.com/stella-emu/stella/blob/76914ded629db887ef612b1e5c9889220808191a/src/emucore/ControllerDetector.cxx
//
// Stella is licenced under the GNU General Public License as published by the
// Free Software Foundation, version 2 or any later version.
//
// https://github.com/stella-emu/stella/blob/76914ded629db887ef612b1e5c9889220808191a/Copyright.txt
func Fingerprint(port plugging.PortID, loader cartridgeloader.Loader) ports.NewPeripheral {
	if port != plugging.PortRight && port != plugging.PortLeft {
		panic(fmt.Sprintf("cannot fingerprint for port %v", port))
	}

	// atarivox and savekey are the most specific peripheral. because atarivox
	// includes the functionality of savekey we need to check atarivox first
	if fingerprintAtariVox(port, loader) {
		return atarivox.NewAtariVox
	}

	if fingerprintSaveKey(port, loader) {
		return savekey.NewSaveKey
	}

	// the other peripherals require a process of differentiation. the order is
	// important.
	if fingerprintStick(port, loader) {
		if fingerprintKeypad(port, loader) {
			return controllers.NewKeypad
		}

		if fingerprintGamepad(port, loader) {
			return controllers.NewGamepad
		}
	} else {
		if fingerprintPaddle(port, loader) {
			return controllers.NewPaddlePair
		}
	}

	// default to normal joystick
	return controllers.NewStick
}

func matchPattern(patterns [][]byte, loader cartridgeloader.Loader) bool {
	for _, p := range patterns {
		if loader.Contains(p) {
			return true
		}
	}

	return false
}

func fingerprintSaveKey(port plugging.PortID, loader cartridgeloader.Loader) bool {
	if port != plugging.PortRight {
		return false
	}

	patterns := [][]byte{
		{ // from I2C_START (i2c.inc)
			0xa9, 0x08, // lda #I2C_SCL_MASK
			0x8d, 0x80, 0x02, // sta SWCHA
			0xa9, 0x0c, // lda #I2C_SCL_MASK|I2C_SDA_MASK
			0x8d, 0x81, // sta SWACNT
		},
		{ // from I2C_START (i2c_v2.1..3.inc)
			0xa9, 0x18, // #(I2C_SCL_MASK|I2C_SDA_MASK)*2
			0x8d, 0x80, 0x02, // sta SWCHA
			0x4a,             // lsr
			0x8d, 0x81, 0x02, // sta SWACNT
		},
		{ // from I2C_START (Strat-O-Gems)
			0xa2, 0x08, // ldx #I2C_SCL_MASK
			0x8e, 0x80, 0x02, // stx SWCHA
			0xa2, 0x0c, // ldx #I2C_SCL_MASK|I2C_SDA_MASK
			0x8e, 0x81, // stx SWACNT
		},
		{ // from I2C_START (AStar, Fall Down, Go Fish!)
			0xa9, 0x08, // lda #I2C_SCL_MASK
			0x8d, 0x80, 0x02, // sta SWCHA
			0xea,       // nop
			0xa9, 0x0c, // lda #I2C_SCL_MASK|I2C_SDA_MASK
			0x8d, // sta SWACNT
		},
	}

	return matchPattern(patterns, loader)
}

func fingerprintAtariVox(port plugging.PortID, loader cartridgeloader.Loader) bool {
	if port != plugging.PortRight {
		return false
	}

	patterns := [][]byte{
		{ // from SPKOUT (speakjet.inc)
			0xad, 0x80, 0x02, // lda SWCHA
			0x29, 0x02, // and #SERIAL_RDYMASK
			0xf0, // beq ...
		},
	}

	if matchPattern(patterns, loader) {
		patterns := [][]byte{
			{ // from SPKOUT (speakjet.inc)
				0x49, 0xff, // eor #$ff
				0xf0, // beq ...
			},
		}

		return matchPattern(patterns, loader)
	}

	return false
}

func fingerprintStick(port plugging.PortID, loader cartridgeloader.Loader) bool {
	var patterns [][]byte

	switch port {
	case plugging.PortLeft:
		patterns = [][]byte{
			{0x24, 0x0c, 0x10},             // bit INPT4; bpl (joystick games only)
			{0x24, 0x0c, 0x30},             // bit INPT4; bmi (joystick games only)
			{0xa5, 0x0c, 0x10},             // lda INPT4; bpl (joystick games only)
			{0xa5, 0x0c, 0x30},             // lda INPT4; bmi (joystick games only)
			{0xb5, 0x0c, 0x10},             // lda INPT4,x; bpl (joystick games only)
			{0xb5, 0x0c, 0x30},             // lda INPT4,x; bmi (joystick games only)
			{0x24, 0x3c, 0x10},             // bit INPT4|$30; bpl (joystick games + Compumate)
			{0x24, 0x3c, 0x30},             // bit INPT4|$30; bmi (joystick, keyboard and mindlink games)
			{0xa5, 0x3c, 0x10},             // lda INPT4|$30; bpl (joystick and keyboard games)
			{0xa5, 0x3c, 0x30},             // lda INPT4|$30; bmi (joystick, keyboard and mindlink games)
			{0xb5, 0x3c, 0x10},             // lda INPT4|$30,x; bpl (joystick, keyboard and driving games)
			{0xb5, 0x3c, 0x30},             // lda INPT4|$30,x; bmi (joystick and keyboard games)
			{0xb4, 0x0c, 0x30},             // ldy INPT4|$30,x; bmi (joystick games only)
			{0xa5, 0x3c, 0x2a},             // ldy INPT4|$30; rol (joystick games only)
			{0xa6, 0x3c, 0x8e},             // ldx INPT4|$30; stx (joystick games only)
			{0xa6, 0x0c, 0x8e},             // ldx INPT4; stx (joystick games only)
			{0xa4, 0x3c, 0x8c},             // ldy INPT4; sty (joystick games only, Scramble)
			{0xa5, 0x0c, 0x8d},             // lda INPT4; sta (joystick games only, Super Cobra Arcade)
			{0xa4, 0x0c, 0x30},             // ldy INPT4|; bmi (only Game of Concentration)
			{0xa4, 0x3c, 0x30},             // ldy INPT4|$30; bmi (only Game of Concentration)
			{0xa5, 0x0c, 0x25},             // lda INPT4; and (joystick games only)
			{0xa6, 0x3c, 0x30},             // ldx INPT4|$30; bmi (joystick games only)
			{0xa6, 0x0c, 0x30},             // ldx INPT4; bmi
			{0xa5, 0x0c, 0x0a},             // lda INPT4; asl (joystick games only)
			{0xb9, 0x0c, 0x00, 0x10},       // lda INPT4,y; bpl (joystick games only)
			{0xb9, 0x0c, 0x00, 0x30},       // lda INPT4,y; bmi (joystick games only)
			{0xb9, 0x3c, 0x00, 0x10},       // lda INPT4,y; bpl (joystick games only)
			{0xb9, 0x3c, 0x00, 0x30},       // lda INPT4,y; bmi (joystick games only)
			{0xa5, 0x0c, 0x0a, 0xb0},       // lda INPT4; asl; bcs (joystick games only)
			{0xb5, 0x0c, 0x29, 0x80},       // lda INPT4,x; and #$80 (joystick games only)
			{0xb5, 0x3c, 0x29, 0x80},       // lda INPT4|$30,x; and #$80 (joystick games only)
			{0xa5, 0x0c, 0x29, 0x80},       // lda INPT4; and #$80 (joystick games only)
			{0xa5, 0x3c, 0x29, 0x80},       // lda INPT4|$30; and #$80 (joystick games only)
			{0xa5, 0x0c, 0x25, 0x0d, 0x10}, // lda INPT4; and INPT5; bpl (joystick games only)
			{0xa5, 0x0c, 0x25, 0x0d, 0x30}, // lda INPT4; and INPT5; bmi (joystick games only)
			{0xa5, 0x3c, 0x25, 0x3d, 0x10}, // lda INPT4|$30; and INPT5|$30; bpl (joystick games only)
			{0xa5, 0x3c, 0x25, 0x3d, 0x30}, // lda INPT4|$30; and INPT5|$30; bmi (joystick games only)
			{0xb5, 0x38, 0x29, 0x80, 0xd0}, // lda INPT0|$30,y; and #$80; bne (Basic Programming)
			{0xa9, 0x80, 0x24, 0x0c, 0xd0}, // lda #$80; bit INPT4; bne (bBasic)
			{0xa5, 0x0c, 0x29, 0x80, 0xd0}, // lda INPT4; and #$80; bne (joystick games only)
			{0xa5, 0x3c, 0x29, 0x80, 0xd0}, // lda INPT4|$30; and #$80; bne (joystick games only)
			{0xad, 0x0c, 0x00, 0x29, 0x80}, // lda.w INPT4|$30; and #$80 (joystick games only)
		}
	case plugging.PortRight:
		patterns = [][]byte{
			{0x24, 0x0d, 0x10},             // bit INPT5; bpl (joystick games only)
			{0x24, 0x0d, 0x30},             // bit INPT5; bmi (joystick games only)
			{0xa5, 0x0d, 0x10},             // lda INPT5; bpl (joystick games only)
			{0xa5, 0x0d, 0x30},             // lda INPT5; bmi (joystick games only)
			{0xb5, 0x0c, 0x10},             // lda INPT4,x; bpl (joystick games only)
			{0xb5, 0x0c, 0x30},             // lda INPT4,x; bmi (joystick games only)
			{0x24, 0x3d, 0x10},             // bit INPT5|$30; bpl (joystick games, Compumate)
			{0x24, 0x3d, 0x30},             // bit INPT5|$30; bmi (joystick and keyboard games)
			{0xa5, 0x3d, 0x10},             // lda INPT5|$30; bpl (joystick games only)
			{0xa5, 0x3d, 0x30},             // lda INPT5|$30; bmi (joystick and keyboard games)
			{0xb5, 0x3c, 0x10},             // lda INPT4|$30,x; bpl (joystick, keyboard and driving games)
			{0xb5, 0x3c, 0x30},             // lda INPT4|$30,x; bmi (joystick and keyboard games)
			{0xa4, 0x3d, 0x30},             // ldy INPT5; bmi (only Game of Concentration)
			{0xa5, 0x0d, 0x25},             // lda INPT5; and (joystick games only)
			{0xa6, 0x3d, 0x30},             // ldx INPT5|$30; bmi (joystick games only)
			{0xa6, 0x0d, 0x30},             // ldx INPT5; bmi
			{0xb9, 0x0c, 0x00, 0x10},       // lda INPT4,y; bpl (joystick games only)
			{0xb9, 0x0c, 0x00, 0x30},       // lda INPT4,y; bmi (joystick games only)
			{0xb9, 0x3c, 0x00, 0x10},       // lda INPT4,y; bpl (joystick games only)
			{0xb9, 0x3c, 0x00, 0x30},       // lda INPT4,y; bmi (joystick games only)
			{0xb5, 0x0c, 0x29, 0x80},       // lda INPT4,x; and #$80 (joystick games only)
			{0xb5, 0x3c, 0x29, 0x80},       // lda INPT4|$30,x; and #$80 (joystick games only)
			{0xa5, 0x3d, 0x29, 0x80},       // lda INPT5|$30; and #$80 (joystick games only)
			{0xb5, 0x38, 0x29, 0x80, 0xd0}, // lda INPT0|$30,y; and #$80; bne (Basic Programming)
			{0xa9, 0x80, 0x24, 0x0d, 0xd0}, // lda #$80; bit INPT5; bne (bBasic)
			{0xad, 0x0d, 0x00, 0x29, 0x80}, // lda.w INPT5|$30; and #$80 (joystick games only)
		}
	}

	return matchPattern(patterns, loader)
}

func fingerprintKeypad(port plugging.PortID, loader cartridgeloader.Loader) bool {
	var patterns [][]byte

	switch port {
	case plugging.PortLeft:
		patterns = [][]byte{
			{0x24, 0x38, 0x30},             // bit INPT0|$30; bmi
			{0xa5, 0x38, 0x10},             // lda INPT0|$30; bpl
			{0xa4, 0x38, 0x30},             // ldy INPT0|$30; bmi
			{0xb5, 0x38, 0x30},             // lda INPT0|$30,x; bmi
			{0x24, 0x08, 0x30},             // bit INPT0; bmi
			{0xa6, 0x08, 0x30},             // ldx INPT0; bmi
			{0xb5, 0x38, 0x29, 0x80, 0xd0}, // lda INPT0,x; and #80; bne
		}

		// keypad fingerprinting is slightly different to the other fingerprint
		// functions in that any matched pattern from the list above is ANDed
		// with a pattern with the list below
		if matchPattern(patterns, loader) {
			patterns = [][]byte{
				{0x24, 0x39, 0x10},             // bit INPT1|$30; bpl
				{0x24, 0x39, 0x30},             // bit INPT1|$30; bmi
				{0xa5, 0x39, 0x10},             // lda INPT1|$30; bpl
				{0xa4, 0x39, 0x30},             // ldy INPT1|$30; bmi
				{0xb5, 0x38, 0x30},             // lda INPT0|$30,x; bmi
				{0x24, 0x09, 0x30},             // bit INPT1; bmi
				{0xa6, 0x09, 0x30},             // ldx INPT1; bmi
				{0xb5, 0x38, 0x29, 0x80, 0xd0}, // lda INPT0,x; and #80; bne
			}
			return matchPattern(patterns, loader)
		}

	case plugging.PortRight:
		patterns = [][]byte{
			{0x24, 0x3a, 0x30},             // bit INPT2|$30; bmi
			{0xa5, 0x3a, 0x10},             // lda INPT2|$30; bpl
			{0xa4, 0x3a, 0x30},             // ldy INPT2|$30; bmi
			{0x24, 0x0a, 0x30},             // bit INPT2; bmi
			{0x24, 0x0a, 0x10},             // bit INPT2; bpl
			{0xa6, 0x0a, 0x30},             // ldx INPT2; bmi
			{0xb5, 0x38, 0x29, 0x80, 0xd0}, // lda INPT2,x; and #80; bne
		}

		// see comment above
		if matchPattern(patterns, loader) {
			patterns = [][]byte{
				{0x24, 0x3b, 0x30},             // bit INPT3|$30; bmi
				{0xa5, 0x3b, 0x10},             // lda INPT3|$30; bpl
				{0xa4, 0x3b, 0x30},             // ldy INPT3|$30; bmi
				{0x24, 0x0b, 0x30},             // bit INPT3; bmi
				{0x24, 0x0b, 0x10},             // bit INPT3; bpl
				{0xa6, 0x0b, 0x30},             // ldx INPT3; bmi
				{0xb5, 0x38, 0x29, 0x80, 0xd0}, // lda INPT2,x; and #80; bne
			}
			return matchPattern(patterns, loader)
		}
	}

	return false
}

func fingerprintGamepad(port plugging.PortID, loader cartridgeloader.Loader) bool {
	var patterns [][]byte

	switch port {
	case plugging.PortLeft:
		patterns = [][]byte{
			{0x24, 0x09, 0x10}, // bit INPT1; bpl (Genesis only)
			{0x24, 0x09, 0x30}, // bit INPT1; bmi (paddle ROMS too)
			{0xa5, 0x09, 0x10}, // lda INPT1; bpl (paddle ROMS too)
			{0xa5, 0x09, 0x30}, // lda INPT1; bmi (paddle ROMS too)
			{0xa4, 0x09, 0x30}, // ldy INPT1; bmi (Genesis only)
			{0xa6, 0x09, 0x30}, // ldx INPT1; bmi (Genesis only)
			{0x24, 0x39, 0x10}, // bit INPT1|$30; bpl (keyboard and paddle ROMS too)
			{0x24, 0x39, 0x30}, // bit INPT1|$30; bmi (keyboard and paddle ROMS too)
			{0xa5, 0x39, 0x10}, // lda INPT1|$30; bpl (keyboard ROMS too)
			{0xa5, 0x39, 0x30}, // lda INPT1|$30; bmi (keyboard and paddle ROMS too)
			{0xa4, 0x39, 0x30}, // ldy INPT1|$30; bmi (keyboard ROMS too)
			{0xa5, 0x39, 0x6a}, // lda INPT1|$30; ror (Genesis only)
			{0xa6, 0x39, 0x8e}, // ldx INPT1|$30; stx (Genesis only)
			{0xa6, 0x09, 0x8e}, // ldx INPT1; stx (Genesis only)
			{0xa4, 0x39, 0x8c}, // ldy INPT1|$30; sty (Genesis only, Scramble)
			{0xa5, 0x09, 0x8d}, // lda INPT1; sta (Genesis only, Super Cobra Arcade)
			{0xa5, 0x09, 0x29}, // lda INPT1; and (Genesis only)
			{0x25, 0x39, 0x30}, // and INPT1|$30; bmi (Genesis only)
			{0x25, 0x09, 0x10}, // and INPT1; bpl (Genesis only)
		}
	case plugging.PortRight:
		patterns = [][]byte{
			{0x24, 0x0b, 0x10}, // bit INPT3; bpl
			{0x24, 0x0b, 0x30}, // bit INPT3; bmi
			{0xa5, 0x0b, 0x10}, // lda INPT3; bpl
			{0xa5, 0x0b, 0x30}, // lda INPT3; bmi
			{0x24, 0x3b, 0x10}, // bit INPT3|$30; bpl
			{0x24, 0x3b, 0x30}, // bit INPT3|$30; bmi
			{0xa5, 0x3b, 0x10}, // lda INPT3|$30; bpl
			{0xa5, 0x3b, 0x30}, // lda INPT3|$30; bmi
			{0xa6, 0x3b, 0x8e}, // ldx INPT3|$30; stx
			{0x25, 0x0b, 0x10}, // and INPT3; bpl (Genesis only)
		}
	}

	return matchPattern(patterns, loader)
}

func fingerprintPaddle(port plugging.PortID, loader cartridgeloader.Loader) bool {
	var patterns [][]byte

	switch port {
	case plugging.PortLeft:
		patterns = [][]byte{
			//{ 0x24, 0x08, 0x10 }, // bit INPT0; bpl (many joystick games too!)
			//{ 0x24, 0x08, 0x30 }, // bit INPT0; bmi (joystick games: Spike's Peak, Sweat, Turbo!)
			{0xa5, 0x08, 0x10}, // lda INPT0; bpl (no joystick games)
			{0xa5, 0x08, 0x30}, // lda INPT0; bmi (no joystick games)
			//{ 0xb5, 0x08, 0x10 }, // lda INPT0,x; bpl (Duck Attack (graphics)!, Toyshop Trouble (Easter Egg))
			{0xb5, 0x08, 0x30},             // lda INPT0,x; bmi (no joystick games)
			{0x24, 0x38, 0x10},             // bit INPT0|$30; bpl (no joystick games)
			{0x24, 0x38, 0x30},             // bit INPT0|$30; bmi (no joystick games)
			{0xa5, 0x38, 0x10},             // lda INPT0|$30; bpl (no joystick games)
			{0xa5, 0x38, 0x30},             // lda INPT0|$30; bmi (no joystick games)
			{0xb5, 0x38, 0x10},             // lda INPT0|$30,x; bpl (Circus Atari, old code!)
			{0xb5, 0x38, 0x30},             // lda INPT0|$30,x; bmi (no joystick games)
			{0x68, 0x48, 0x10},             // pla; pha; bpl (i.a. Bachelor Party)
			{0xa5, 0x08, 0x4c},             // lda INPT0; jmp (only Backgammon)
			{0xa4, 0x38, 0x30},             // ldy INPT0; bmi (no joystick games)
			{0xb9, 0x08, 0x00, 0x30},       // lda INPT0,y; bmi (i.a. Encounter at L-5)
			{0xb9, 0x38, 0x00, 0x30},       // lda INPT0|$30,y; bmi (i.a. SW-Jedi Arena, Video Olympics)
			{0xb9, 0x08, 0x00, 0x10},       // lda INPT0,y; bpl (Drone Wars)
			{0x24, 0x08, 0x30, 0x02},       // bit INPT0; bmi +2 (Picnic)
			{0xb5, 0x38, 0x29, 0x80, 0xd0}, // lda INPT0|$30,x; and #$80; bne (Basic Programming)
			{0x24, 0x38, 0x85, 0x08, 0x10}, // bit INPT0|$30; sta COLUPF, bpl (Fireball)
			{0xb5, 0x38, 0x49, 0xff, 0x0a}, // lda INPT0|$30,x; eor #$ff; asl (Blackjack)
			{0xb1, 0xf2, 0x30, 0x02, 0xe6}, // lda ($f2),y; bmi...; inc (Warplock)
		}
	case plugging.PortRight:
		patterns = [][]byte{
			{0x24, 0x0a, 0x10},             // bit INPT2; bpl (no joystick games)
			{0x24, 0x0a, 0x30},             // bit INPT2; bmi (no joystick games)
			{0xa5, 0x0a, 0x10},             // lda INPT2; bpl (no joystick games)
			{0xa5, 0x0a, 0x30},             // lda INPT2; bmi
			{0xb5, 0x0a, 0x10},             // lda INPT2,x; bpl
			{0xb5, 0x0a, 0x30},             // lda INPT2,x; bmi
			{0xb5, 0x08, 0x10},             // lda INPT0,x; bpl (no joystick games)
			{0xb5, 0x08, 0x30},             // lda INPT0,x; bmi (no joystick games)
			{0x24, 0x3a, 0x10},             // bit INPT2|$30; bpl
			{0x24, 0x3a, 0x30},             // bit INPT2|$30; bmi
			{0xa5, 0x3a, 0x10},             // lda INPT2|$30; bpl
			{0xa5, 0x3a, 0x30},             // lda INPT2|$30; bmi
			{0xb5, 0x3a, 0x10},             // lda INPT2|$30,x; bpl
			{0xb5, 0x3a, 0x30},             // lda INPT2|$30,x; bmi
			{0xb5, 0x38, 0x10},             // lda INPT0|$30,x; bpl  (Circus Atari, old code!)
			{0xb5, 0x38, 0x30},             // lda INPT0|$30,x; bmi (no joystick games, except G.I. Joe)
			{0xa4, 0x3a, 0x30},             // ldy INPT2|$30; bmi (no joystick games)
			{0xa5, 0x3b, 0x30},             // lda INPT3|$30; bmi (only Tac Scan, ports and paddles swapped)
			{0xb9, 0x38, 0x00, 0x30},       // lda INPT0|$30,y; bmi (Video Olympics)
			{0xb5, 0x38, 0x29, 0x80, 0xd0}, // lda INPT0|$30,x; and #$80; bne (Basic Programming)
			{0x24, 0x38, 0x85, 0x08, 0x10}, // bit INPT2|$30; sta COLUPF, bpl (Fireball, patched at runtime!)
			{0xb5, 0x38, 0x49, 0xff, 0x0a}, // lda INPT0|$30,x; eor #$ff; asl (Blackjack)
		}
	}

	return matchPattern(patterns, loader)
}
