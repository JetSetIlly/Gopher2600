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
// The FontAwesome font (fa-solid-900.ttf) was downloaded on 18th March 2020
// from https://fontawesome.com/download using the "Free for Web" button. Full
// URL was:
//
// https://use.fontawesome.com/releases/v5.15.2/fontawesome-free-5.15.2-web.zip
//
// FontAwesome is licenced under the Font Awesome Free License.
package fonts

import _ "embed"

//go:embed "fa-solid-900.ttf"
var FontAwesome []byte

// Unicode points in FontAwesome for icons used in the application.
const (
	Run             = '\uf04b'
	Halt            = '\uf04c'
	Back            = '\uf053'
	Disk            = '\uf0c7'
	Mouse           = '\uf8cc'
	GoingForward    = '\uf01e'
	Persist         = '\uf021'
	Breakpoint      = '\uf12a'
	AudioDisabled   = '\uf00d'
	AudioEnabled    = '\uf028'
	TermPrompt      = '\uf105'
	ColorSwatch     = '\uf111'
	TapeRewind      = '\uf049'
	TapePlay        = '\uf04b'
	TapeStop        = '\uf04d'
	TapeFastForward = '\uf04e'
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
	Stick    = '\ue000'
	Paddle   = '\ue001'
	Keyboard = '\ue002'
	Tape     = '\ue003'
)

// The first and last unicode points used in the application. We use this to
// make sure we're using as small a font texture as possible.
const (
	Gopher2600IconMin = '\ue000'
	Gopher2600IconMax = '\ue003'
)
