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

// Package fonts embeds a set of FontAwesome glyphs as font data into a byte
// array. This can then be used with dear imgui's AddFontFromMemoryTTF() or
// similar, to merge the icons with the default font palette. These icons can
// be used alongside regular text for an inline icon.
//
// Gopher2600Icons meanwhile is a sparse set of font data containing icons that
// are intended to be shown individually (ie. without accopanying text).
//
// Image for the Controller icons taken from Wikipedia. Reduced in size to 256
// pixel width; Converted to SVG with the help of Inkscape's Trace Bitmap
// function; and finally imported into an empty TTF file using FontForge.
//
// Licencing
//
// Gopher2600-Icons.ttf is licenced by Stephen Illingworth, under the Creative
// Commons Attribution 4.0 International licence.
//
// https://creativecommons.org/licenses/by/4.0/legalcode
//
//
// The FontAwesome font (fa-solid-900.ttf) was downloaded on 18th March 2020
// from https://fontawesome.com/download using the "Free for Web" button. Full
// URL was:
//
// https://use.fontawesome.com/releases/v5.15.2/fontawesome-free-5.15.2-web.zip
//
// FontAwesome is licenced under the Font Awesome Free License.
//
//
// Hack-Regular was downloaded on 20th December 2021 from permalink URL:
//
// https://github.com/source-foundry/Hack/blob/a737c121cabb337fdfe655d8c7304729f351e30f/build/ttf/Hack-Regular.ttf
//
// Hack-Regular is licenced under the MIT License.
package fonts

import _ "embed"

//go:embed "fa-solid-900.ttf"
var FontAwesome []byte

// Unicode points in FontAwesome for icons used in the application.
const (
	Run                    = '\uf04b'
	Halt                   = '\uf04c'
	BackClock              = '\uf104'
	BackInstruction        = '\uf100'
	BackScanline           = '\uf106'
	BackFrame              = '\uf102'
	StepOver               = '\uf2f9'
	Disk                   = '\uf0c7'
	Mouse                  = '\uf8cc'
	GoingForward           = '\uf01e'
	Persist                = '\uf021'
	Breakpoint             = '\uf12a'
	AudioDisabled          = '\uf026'
	AudioEnabled           = '\uf028'
	TermPrompt             = '\uf105'
	ColorSwatch            = '\uf111'
	TapeRewind             = '\uf049'
	TapePlay               = '\uf04b'
	TapeStop               = '\uf04d'
	TapeFastForward        = '\uf04e'
	EmulationPause         = '\uf04c'
	EmulationRun           = '\uf04b'
	EmulationRewindBack    = '\uf04a'
	EmulationRewindForward = '\uf04e'
	EmulationRewindAtStart = '\uf049'
	EmulationRewindAtEnd   = '\uf050'
	MusicNote              = '\uf001'
	VolumeUp               = '\uf062'
	VolumeDown             = '\uf063'
	Camera                 = '\uf030'
	Chip                   = '\uf2db'
	Unlocked               = '\uf13e'
	Bug                    = '\uf188'
)

// The first and last unicode points used in the application. We use this to
// make sure we're using as small a font texture as possible.
const (
	FontAwesomeMin = '\ue005'
	FontAwesomeMax = '\uf8ff'
)

//go:embed "Gopher2600-Icons.ttf"
var Gopher2600Icons []byte

// Unicode points in AtariIcons for icons used in the application.
const (
	Stick   = '\ue000'
	Paddle  = '\ue001'
	Keypad  = '\ue002'
	Tape    = '\ue003'
	Wifi    = '\ue004'
	Savekey = '\ue005'
)

// The first and last unicode points used in the application. We use this to
// make sure we're using as small a font texture as possible.
const (
	Gopher2600IconMin = '\ue000'
	Gopher2600IconMax = '\ue005'
)

//go:embed "Hack-Regular.ttf"
var Hack []byte

// The first and last unicode points used in the application. We use this to
// make sure we're using as small a font texture as possible.
const (
	HackMin = '\u0003'
	HackMax = '\u1ef9'
)
