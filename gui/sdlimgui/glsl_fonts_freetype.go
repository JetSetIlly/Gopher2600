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
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/gui/fonts"
)

func setDefaultFont() (imgui.FontAtlas, bool, float32, error) {
	atlas := imgui.CurrentIO().Fonts()
	atlas.SetFontBuilderFlags(imgui.FreeTypeBuilderFlagsForceAutoHint)

	// load jetbrains mono font
	cfg := imgui.NewFontConfig()
	defer cfg.Delete()
	cfg.SetPixelSnapH(true)

	var builder imgui.GlyphRangesBuilder
	builder.Add(fonts.JetBrainsMonoMin, fonts.JetBrainsMonoMax)

	size := float32(14.0)
	font := atlas.AddFontFromMemoryTTFV(fonts.JetBrainsMono, size, cfg, builder.Build().GlyphRanges)
	if font == 0 {
		return atlas, true, size, curated.Errorf("font: error loading jetBrainsMono font from memory")
	}

	return atlas, true, size, nil
}
