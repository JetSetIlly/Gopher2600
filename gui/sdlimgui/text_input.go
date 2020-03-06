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
	"strings"

	"github.com/inkyblackness/imgui-go/v2"
)

// calls imguiInput with the string of allowed hexadecimal characters.
func imguiHexInput(label string, aggressiveUpdate bool, digits int, content *string) bool {
	return imguiInput(label, aggressiveUpdate, digits, content, "abcdefABCDEF0123456789")
}

// input text that accepts a maximum number of hex digits. physical width of
// InpuText should be controlled with PushItemWidth()/PopItemWidth() as normal.
func imguiInput(label string, aggressiveUpdate bool, digits int, content *string, allowedChars string) bool {
	cb := func(d imgui.InputTextCallbackData) int32 {
		switch d.EventFlag() {
		case imgui.InputTextFlagsCallbackCharFilter:
			// filter characters that are not in the list of allowedChars
			if !strings.ContainsAny(string(d.EventChar()), allowedChars) {
				return -1
			}
		default:
			b := string(d.Buffer())

			// restrict length of input to two characters. note that restriction to
			// hexadecimal characters is handled by imgui's CharsHexadecimal flag
			// given to InputTextV()
			if len(b) > digits {
				d.DeleteBytes(0, len(b))
				b = b[:digits]
				d.InsertBytes(0, []byte(b))
				d.MarkBufferModified()
			}
		}

		return 0
	}

	// flags used with InputTextV(). not using InputTextFlagsCharsHexadecimal
	// and preferring to filter manually for greated flexibility
	flags := imgui.InputTextFlagsCallbackCharFilter |
		imgui.InputTextFlagsCallbackAlways |
		imgui.InputTextFlagsAutoSelectAll

	// with aggressiveUpdate the values entered will be given to the onEnter()
	// function immediately and not just when the enter key is pressed.
	if aggressiveUpdate {
		flags |= imgui.InputTextFlagsEnterReturnsTrue
	}

	return imgui.InputTextV(label, content, flags, cb)
}
