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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/sahilm/fuzzy"
)

type fuzzyMatcher struct {
	// whether the input text has been selected. if it is active then the
	// listbox is shown
	active bool

	// what has been typed in the input text
	typed string

	// the last fuzzy match results. if the list is empty the raw choices is
	// used
	matches      fuzzy.Matches
	matchedInput string

	// controls when to refocus the keyboard on the input text
	reactivateInput bool

	// currently selected list entry. changed with the up/down keys via the
	// InpuText callback function
	selected int
}

func (fz *fuzzyMatcher) textInput(label string, content string, choices []string, onChange func(string)) {
	imgui.PushItemWidth(-1)
	imgui.BeginGroup()

	flags := imgui.InputTextFlagsEnterReturnsTrue
	flags |= imgui.InputTextFlagsCallbackAlways
	flags |= imgui.InputTextFlagsCallbackHistory
	flags |= imgui.InputTextFlagsCallbackCharFilter

	// simply display content if InputText is not active
	if !fz.active {
		fz.typed = content
	}

	cb := func(d imgui.InputTextCallbackData) int32 {
		// we want to delete the content on first activation
		if !fz.active {
			d.DeleteBytes(0, len(d.Buffer()))
		}

		// use the InputText history callback to adjust the selected item in
		// the ListBox
		switch d.EventFlag() {
		case imgui.InputTextFlagsCallbackHistory:
			switch d.EventKey() {
			case imgui.KeyUpArrow:
				if fz.selected > 0 {
					fz.selected--
				}
			case imgui.KeyDownArrow:
				if len(fz.matches) == 0 {
					if fz.selected < len(choices)-1 {
						fz.selected++
					}
				} else {
					if fz.selected < len(fz.matches)-1 {
						fz.selected++
					}
				}
			}
		case imgui.InputTextFlagsCallbackCharFilter:
			fz.selected = 0
		}

		return 0
	}

	// call onChange() when return is pressed. onChange() will also be called
	// if an entry in the listbox is selected
	if imgui.InputTextV(label, &fz.typed, flags, cb) {
		if len(fz.matches) == 0 {
			onChange(choices[fz.selected])
		} else {
			onChange(fz.matches[fz.selected].Str)
		}
		fz.active = false
	}

	// set keyboard focus on InputText
	if fz.reactivateInput {
		imgui.SetKeyboardFocusHereV(-1)
		fz.reactivateInput = false
	}

	if imgui.IsItemActivated() {
		fz.active = true
	}

	if fz.active {
		// information about how many matches there are. this is displayed
		// after the listbox is drawn
		var footer string

		if imgui.BeginListBox(fmt.Sprintf("%s##popup", label)) {
			// the content of the listbox is made up of one of either two
			// lists. the drawOption() function is in charge of drawing the
			// selectable
			drawOption := func(i int, s string) {
				if i == fz.selected {
					imgui.SetScrollHereY(0.0)
				}
				if imgui.SelectableV(s, i == fz.selected, imgui.SelectableFlagsNone, imgui.Vec2{}) {
					onChange(s)
					fz.active = false
				}
			}

			// if nothing has been typed in the input box then simply display
			// all the choices. otherwise we show the fuzzy match results
			if fz.typed == "" {
				for i, s := range choices {
					drawOption(i, s)
				}
			} else {
				// run fuzzy matcher here if input has changed
				if fz.typed != fz.matchedInput {
					fz.matchedInput = fz.typed
					fz.matches = fuzzy.Find(fz.typed, choices)
				}

				for i, m := range fz.matches {
					drawOption(i, m.Str)
				}

				switch len(fz.matches) {
				case 0:
					footer = "no matches"
				case 1:
					footer = "1 match"
				default:
					footer = fmt.Sprintf("%d matches", len(fz.matches))
				}
			}
			imgui.EndListBox()

			if footer != "" {
				imgui.Text(footer)
			}
		}

		// reactivate input text when interaction with listbox ends but not if
		// an entry has been clicked (fz.active will be false if it has)
		if fz.active && imgui.IsItemDeactivated() {
			fz.reactivateInput = true
		}
	}

	imgui.EndGroup()
	imgui.PopItemWidth()

	if imgui.IsMouseClicked(0) && !imgui.IsItemHovered() {
		fz.active = false
	}
}
