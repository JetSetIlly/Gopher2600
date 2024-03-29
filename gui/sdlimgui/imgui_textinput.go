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

	"github.com/inkyblackness/imgui-go/v4"
)

// length limited input of allowed hexadecimal characters.
func imguiHexInput(label string, length int, content *string) bool {
	return imguiInputLenLimited(label, length, content, "abcdefABCDEF0123456789", true)
}

// length limited input of numeric characters.
func imguiDecimalInput(label string, length int, content *string) bool {
	return imguiInputLenLimited(label, length, content, "0123456789", true)
}

// length limited input of numeric characters for floatingPoint
func imguiFloatingPointInput(label string, length int, content *string) bool {
	return imguiInputLenLimited(label, length, content, "0123456789.", true)
}

// length limited input of text characters.
func imguiTextInput(label string, length int, content *string, selectAll bool) bool {
	return imguiInputLenLimited(label, length, content, "", selectAll)
}

// input text that accepts a maximum number of characters.
//
// if allowedChars is the empty string than all characters will be allowed.
func imguiInputLenLimited(label string, length int, content *string, allowedChars string, selectAll bool) bool {
	cb := func(d imgui.InputTextCallbackData) int32 {
		switch d.EventFlag() {
		case imgui.InputTextFlagsCallbackCharFilter:
			if allowedChars != "" {
				// filter characters that are not in the list of allowedChars
				if !strings.ContainsAny(string(d.EventChar()), allowedChars) {
					return -1
				}
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

	flags := imgui.InputTextFlagsCallbackCharFilter | imgui.InputTextFlagsCallbackAlways | imgui.InputTextFlagsEnterReturnsTrue

	if selectAll {
		flags |= imgui.InputTextFlagsAutoSelectAll
	}

	imgui.PushItemWidth(imguiTextWidth(length))
	defer imgui.PopItemWidth()

	return imgui.InputTextV(label, content, flags, cb)
}
