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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/sahilm/fuzzy"
)

// filterFlags is used to control how the filter control is drawn
type filterFlags int

// list of valid filterFlags values
const (
	filterFlagsNone     filterFlags = 0
	filterFlagsNoSpaces filterFlags = 1 << iota
)

// filter provides a basic UI control for managing filter text
type filter struct {
	img   *SdlImgui
	flags filterFlags

	text          string
	caseSensitive bool
	fuzzy         bool
}

func newFilter(img *SdlImgui, flags filterFlags) filter {
	return filter{
		img:   img,
		flags: flags,
		fuzzy: true,
	}
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
			if unicode.IsPrint(f.img.phantomInputRune) {
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
		func() {
			if len(f.text) == 0 {
				imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
				imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
				defer imgui.PopStyleVar()
				defer imgui.PopItemFlag()
			}

			imgui.AlignTextToFramePadding()
			filterText := "no filter"
			if len(f.text) > 0 {
				filterText = f.text
			}
			imgui.Text(fmt.Sprintf("%s%s", filterText, strings.Repeat(" ", 15-len(filterText))))

			imgui.SameLine()
			if imgui.Button(string(fonts.Trash)) {
				f.text = ""
			}
		}()

		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()

		imgui.Checkbox("Fuzzy Match", &f.fuzzy)

		func() {
			// if fuzzy matching is enabled that case sensitivity is irrelevant
			if f.fuzzy {
				imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
				imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
				defer imgui.PopStyleVar()
				defer imgui.PopItemFlag()
			}
			imgui.Checkbox("Case Sensitive", &f.caseSensitive)
		}()

		imgui.EndPopup()
	}
}

// isFiltered returns true if the provided string matches the filter text
func (f *filter) isFiltered(s string) bool {
	if f.fuzzy {
		return len(f.text) > 0 && fuzzy.Find(f.text, []string{s}).Len() == 0
	}

	if f.caseSensitive {
		return !strings.HasPrefix(s, f.text)
	}
	return !strings.HasPrefix(strings.ToLower(s), strings.ToLower(f.text))
}
