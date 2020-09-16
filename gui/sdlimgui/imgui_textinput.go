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
	"strings"

	"github.com/inkyblackness/imgui-go/v2"
)

// calls imguiInput with the string of allowed hexadecimal characters.
func imguiHexInput(label string, aggressiveUpdate bool, length int, content *string) bool {
	return imguiInput(label, aggressiveUpdate, length, content, "abcdefABCDEF0123456789")
}

// calls imguiInput with the string of numeric characters.
func imguiDecimalInput(label string, aggressiveUpdate bool, length int, content *string) bool {
	return imguiInput(label, aggressiveUpdate, length, content, "0123456789")
}

// input text that accepts a maximum number of characters. physical width of
// InpuText should be controlled with PushItemWidth()/PopItemWidth() as normal.
func imguiInput(label string, aggressiveUpdate bool, length int, content *string, allowedChars string) bool {
	cb := func(d imgui.InputTextCallbackData) int32 {
		switch d.EventFlag() {
		case imgui.InputTextFlagsCallbackCharFilter:
			// filter characters that are not in the list of allowedChars
			if !strings.ContainsAny(string(d.EventChar()), allowedChars) {
				return -1
			}
		default:
			b := string(d.Buffer())

			// restrict length of input
			if len(b) > length {
				d.DeleteBytes(0, len(b))
				b = b[:length]
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
