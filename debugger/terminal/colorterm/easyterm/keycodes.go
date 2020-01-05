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

package easyterm

// list of ASCII codes for non-alphanumeric characters
const (
	KeyInterrupt      = 3  // end-of-text character
	KeySuspend        = 26 // substitute character
	KeyTab            = 9
	KeyCarriageReturn = 13
	KeyEsc            = 27
	KeyBackspace      = 8
)

// list of ASCII code for characters that can follow KeyEsc
const (
	EscDelete = 51
	EscCursor = 91
	EscHome   = 72
	EscEnd    = 70
)

// list of ASCII code for characters that can follow EscCursor
const (
	CursorUp       = 'A'
	CursorDown     = 'B'
	CursorForward  = 'C'
	CursorBackward = 'D'
)
