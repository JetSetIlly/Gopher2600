//go:build imguifreetype

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
	"github.com/jetsetilly/gopher2600/gui/fonts"
)

func (fnts *fontAtlas) isFreeType() bool {
	return true
}

func (fnts *fontAtlas) setDefaultFont(prefs *preferences) error {
	defaultFontSize := float32(prefs.guiFont.Get().(float64))
	if fnts.defaultFont != 0 && defaultFontSize == fnts.defaultFontSize {
		return nil
	}

	atlas := imgui.CurrentIO().Fonts()
	atlas.SetFontBuilderFlags(imgui.FreeTypeBuilderFlagsForceAutoHint)

	// load gui font (default)
	cfg := imgui.NewFontConfig()
	defer cfg.Delete()
	cfg.SetPixelSnapH(true)
	if int(defaultFontSize)%2 == 0.0 {
		cfg.SetGlyphOffsetY(1.0)
	}

	var builder imgui.GlyphRangesBuilder
	builder.Add(fonts.JetBrainsMonoMin, fonts.JetBrainsMonoMax)

	fnts.defaultFontSize = float32(defaultFontSize)
	fnts.defaultFont = atlas.AddFontFromMemoryTTFV(fonts.JetBrainsMono, fnts.defaultFontSize, cfg, builder.Build().GlyphRanges)
	if fnts.defaultFont == 0 {
		return fmt.Errorf("font: error loading JetBrainsMono font from memory")
	}

	fnts.mergeFontAwesome(fnts.defaultFontSize, 1.0)

	return nil
}

func (fnts *fontAtlas) sourceCodeFont(prefs *preferences) error {
	codeSize := float32(prefs.codeFont.Get().(float64))
	if fnts.code != 0 && codeSize == fnts.codeSize {
		return nil
	}

	atlas := imgui.CurrentIO().Fonts()

	cfg := imgui.NewFontConfig()
	defer cfg.Delete()
	cfg.SetPixelSnapH(true)

	var builder imgui.GlyphRangesBuilder
	builder.Add(fonts.JetBrainsMonoMin, fonts.JetBrainsMonoMax)

	fnts.codeSize = codeSize
	fnts.code = atlas.AddFontFromMemoryTTFV(fonts.JetBrainsMono, fnts.codeSize, cfg, builder.Build().GlyphRanges)
	if fnts.code == 0 {
		return fmt.Errorf("font: error loading JetBrainsMono font from memory")
	}

	fnts.mergeFontAwesome(fnts.codeSize, 0.0)

	return nil
}
