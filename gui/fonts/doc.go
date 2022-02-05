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
//
//
// JetBrainsMono-Regular is licenced under the OFL-1.1 License
package fonts
