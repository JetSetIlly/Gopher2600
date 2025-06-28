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
	"fmt"
	"strings"
	"unicode"

	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/imgui-go/v5"
)

// filterFlags is used to control how the filter control is drawn
type filterFlags int

// list of valid filterFlags values
const (
	filterFlagsNone     filterFlags = 0
	filterFlagsNoSpaces filterFlags = 1 << iota
	filterFlagsNoPunctuation
	filterFlagsOnlyASCII
	filterFlagsNoEdit
)

// common filter flags combinations
const (
	filterFlagsVariableNamesC = filterFlagsOnlyASCII | filterFlagsNoSpaces | filterFlagsNoPunctuation
)

// filter provides a basic UI control for managing filter text
type filter struct {
	img   *SdlImgui
	flags filterFlags

	text          string
	caseSensitive bool
	prefixOnly    bool
}

func newFilter(img *SdlImgui, flags filterFlags) filter {
	return filter{
		img:   img,
		flags: flags,
	}
}

func (f *filter) applyRules(r rune) bool {
	return unicode.IsPrint(r) &&
		(f.flags&filterFlagsOnlyASCII != filterFlagsOnlyASCII || r < unicode.MaxASCII) &&
		(f.flags&filterFlagsNoPunctuation != filterFlagsNoPunctuation ||
			unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r)) &&
		(f.flags&filterFlagsNoSpaces != filterFlagsNoSpaces || !unicode.IsSpace(r))
}

// draw the imgui control
func (f *filter) draw(id string) {
	// correct id string if necessary
	if !strings.HasPrefix(id, "##") {
		id = fmt.Sprintf("##%s", id)
	}

	// obey phantom input directive if window or any child windows are focused
	if imgui.IsWindowFocusedV(imgui.FocusedFlagsRootAndChildWindows) {
		switch f.img.phantomInput {
		case phantomInputBackSpace:
			if len(f.text) > 0 {
				f.text = f.text[:len(f.text)-1]
			}
		case phantomInputRune:
			// make sure phaontom input rune is printable and that it obeys the filter rules
			if f.applyRules(f.img.phantomInputRune) {
				f.text = fmt.Sprintf("%s%c", f.text, f.img.phantomInputRune)
			}
		}
	}

	imgui.SameLineV(0, 15)
	if imgui.Button(strings.TrimSpace(fmt.Sprintf("%c %s", fonts.Filter, f.text))) {
		mp := imgui.MousePos()
		imgui.SetNextWindowPos(mp)
		imgui.OpenPopup(id)
	}

	// clear filter if right mouse button is click over filter button
	if imgui.IsItemHovered() && imgui.IsMouseClicked(1) {
		f.text = ""
	}

	if imgui.BeginPopupV(id, imgui.WindowFlagsNoMove) {
		if f.flags&filterFlagsNoEdit == filterFlagsNoEdit {
			// function wrapped for the convenience of the defer statement
			func() {
				drawDisabled(len(f.text) == 0, func() {
					imgui.AlignTextToFramePadding()
					filterText := "no filter"
					if len(f.text) > 0 {
						filterText = f.text
					}
					imgui.Text(fmt.Sprintf("%s%s", filterText, strings.Repeat(" ", 15-len(filterText))))
				})
			}()
		} else {
			imgui.PushItemWidth(10 * imgui.FontSize())

			cb := func(ev imgui.InputTextCallbackData) int32 {
				if f.applyRules(ev.EventChar()) {
					return 0
				}
				return 1
			}
			imgui.InputTextV("", &f.text, imgui.InputTextFlagsCallbackCharFilter, cb)

			imgui.PopItemWidth()
		}

		imgui.SameLine()
		if imgui.Button(string(fonts.Trash)) {
			f.text = ""
		}

		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()

		imgui.Checkbox("Case Sensitive", &f.caseSensitive)
		imgui.Checkbox("Prefix Only", &f.prefixOnly)

		imgui.EndPopup()
	}
}

// isFiltered returns true if the provided string matches the filter text
func (f *filter) isFiltered(s string) bool {
	var test func(string, string) bool

	if f.prefixOnly {
		test = strings.HasPrefix
	} else {
		test = strings.Contains
	}

	if f.caseSensitive {
		return !test(s, f.text)
	}
	return !test(strings.ToLower(s), strings.ToLower(f.text))
}
