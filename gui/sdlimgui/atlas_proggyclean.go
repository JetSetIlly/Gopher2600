//go:build !imguifreetype

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
	"github.com/inkyblackness/imgui-go/v4"
)

func (fnts *fontAtlas) isFreeType() bool {
	return false
}

func (fnts *fontAtlas) setDefaultFont(prefs *preferences) error {
	// default font has already been set up
	if fnts.defaultFont != 0 {
		return nil
	}

	atlas := imgui.CurrentIO().Fonts()
	fnts.defaultFont = atlas.AddFontDefault()
	fnts.defaultFontSize = 13.0
	fnts.mergeFontAwesome(fnts.defaultFontSize, 2.0)
	return nil
}

func (fnts *fontAtlas) sourceCodeFont(prefs *preferences) error {
	fnts.code = fnts.defaultFont
	fnts.codeSize = fnts.defaultFontSize
	return nil
}
