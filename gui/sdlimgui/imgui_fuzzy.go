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

	"github.com/jetsetilly/imgui-go/v5"
	"github.com/sahilm/fuzzy"
)

type fuzzyFilter struct {
	// results are showing
	active bool

	// what has been filter into the filter text box
	filter string

	// controls when to refocus the keyboard on the input text
	refocusKeyboard bool

	// the last fuzzy match results
	matches fuzzy.Matches

	// the filter used for the last fuzzy.Matches result. fuzzy matching will
	// not run again until filter changes
	matchedOnFilter string

	// currently selected list entry. changed with the up/down keys via the
	// InpuText callback function
	selected int

	// scroll to the selected item at the next opportunity
	scrollToSelected bool
}

func (fz *fuzzyFilter) deactivate() {
	fz.active = false
	fz.filter = ""
	fz.refocusKeyboard = false
	fz.matches = fz.matches[:0]
	fz.matchedOnFilter = ""
	fz.selected = 0
	fz.scrollToSelected = false
}

func (fz *fuzzyFilter) activate() {
	fz.active = true
}

func (fz *fuzzyFilter) draw(label string, choices any, onChange func(int), allowEmptyFilter bool) bool {
	if !fz.active {
		fz.active = true
		fz.refocusKeyboard = true
	}

	// textinput and listbox are grouped together
	imgui.BeginGroup()

	// callback function for InputText widget
	cb := func(d imgui.InputTextCallbackData) int32 {
		// use the InputText history callback to adjust the selected item in
		// the ListBox
		switch d.EventFlag() {
		case imgui.InputTextFlagsCallbackHistory:
			switch d.EventKey() {
			case imgui.KeyUpArrow:
				if fz.selected > 0 {
					fz.selected--
					fz.scrollToSelected = true
				}
			case imgui.KeyDownArrow:
				if len(fz.matches) == 0 {
					switch choices := choices.(type) {
					case []string:
						if fz.selected < len(choices)-1 {
							fz.selected++
							fz.scrollToSelected = true
						}
					case fuzzy.Source:
						if fz.selected < choices.Len()-1 {
							fz.selected++
							fz.scrollToSelected = true
						}
					}
				} else {
					if fz.selected < len(fz.matches)-1 {
						fz.selected++
						fz.scrollToSelected = true
					}
				}
			}
		case imgui.InputTextFlagsCallbackCharFilter:
			fz.selected = 0
			fz.scrollToSelected = true
		}

		return 0
	}

	// filter input
	flags := imgui.InputTextFlagsEnterReturnsTrue
	flags |= imgui.InputTextFlagsCallbackAlways
	flags |= imgui.InputTextFlagsCallbackHistory
	flags |= imgui.InputTextFlagsCallbackCharFilter
	if imgui.InputTextV(label, &fz.filter, flags, cb) {
		// call onChange() when return is pressed. note that onChange() will
		// also be called if an entry in the results list is selected
		if len(fz.matches) == 0 {
			if allowEmptyFilter {
				onChange(fz.selected)
			}
		} else {
			onChange(fz.matches[fz.selected].Index)
		}
		fz.deactivate()
	}

	// refocus keyboard onto filter input
	if fz.refocusKeyboard {
		imgui.SetKeyboardFocusHereV(-1)
		fz.refocusKeyboard = false
	}

	// show results listbox
	if imgui.BeginListBox(fmt.Sprintf("%s##popup", label)) {
		// the content of the listbox is made up of one of either two
		// lists. the drawOption() function is in charge of drawing the
		// selectable
		drawOption := func(i int, s string) {
			s = strings.TrimSpace(s)
			if len(s) == 0 {
				return
			}

			if i == fz.selected && fz.scrollToSelected {
				imgui.SetScrollHereY(0.0)
				fz.scrollToSelected = false
			}
			if imgui.SelectableV(s, i == fz.selected, imgui.SelectableFlagsNone, imgui.Vec2{}) {
				// similar logic to when InputTextV() returns true. however,
				// rather than using the fz.selected value we use i which
				// represents the value under the mouse
				if len(fz.matches) == 0 {
					if allowEmptyFilter {
						onChange(i)
					}
				} else {
					onChange(fz.matches[i].Index)
				}

				fz.deactivate()
			}
		}

		// if nothing has been typed in the input box then simply display
		// all the choices. otherwise we show the fuzzy match results
		if fz.filter == "" {
			if allowEmptyFilter {
				switch choices := choices.(type) {
				case []string:
					for i, s := range choices {
						drawOption(i, s)
					}
				case fuzzy.Source:
					for i := 0; i < choices.Len(); i++ {
						drawOption(i, choices.String(i))
					}
				}
			}
		} else {
			// run fuzzy matcher here if input has changed
			if fz.filter != fz.matchedOnFilter {
				fz.matchedOnFilter = fz.filter
				switch choices := choices.(type) {
				case []string:
					fz.matches = fuzzy.Find(fz.filter, choices)
				case fuzzy.Source:
					fz.matches = fuzzy.FindFrom(fz.filter, choices)
				}
			}

			for i, m := range fz.matches {
				drawOption(i, m.Str)
			}
		}
		imgui.EndListBox()

		// summary information about how many matches there are. this is displayed
		// after the listbox is drawn
		switch len(fz.matches) {
		case 0:
			if !allowEmptyFilter {
				imgui.Text("no matches")
			} else {
				switch choices := choices.(type) {
				case []string:
					imgui.Text(fmt.Sprintf("%d entries", len(choices)))
				case fuzzy.Source:
					imgui.Text(fmt.Sprintf("%d entries", choices.Len()))
				}
			}
		case 1:
			imgui.Text("1 match")
		default:
			imgui.Text(fmt.Sprintf("%d matches", len(fz.matches)))
		}
	}

	// reactivate input text when interaction (ie. scrolling) with listbox
	// ends but not if an entry has been clicked (fz.active will be false
	// if it has)
	if fz.active && imgui.IsItemDeactivated() {
		// fz.refocusKeyboard = true
	}

	imgui.EndGroup()

	if !fz.active && imgui.IsItemActivated() {
		// fz.refocusKeyboard = true
		fz.active = true
	}

	// deactivatea fuzzy filter if mouse click occurs outside the group
	if imgui.IsMouseClicked(0) && !imgui.IsItemHovered() {
		fz.deactivate()
	}

	return fz.active
}
