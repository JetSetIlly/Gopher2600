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

type fontAtlas struct {
	// default font
	defaultFont     imgui.Font
	defaultFontSize float32

	// used for notifications (eg. network access, etc.)
	largeFontAwesome     imgui.Font
	largeFontAwesomeSize float32

	// used for rewind/fast-forward state
	veryLargeFontAwesome     imgui.Font
	veryLargeFontAwesomeSize float32

	// custom icons for controllers and other peripherals
	gopher2600Icons     imgui.Font
	gopher2600IconsSize float32

	// annotation of diagrams
	diagram     imgui.Font
	diagramSize float32

	// terminal
	terminal     imgui.Font
	terminalSize float32

	// source code
	code     imgui.Font
	codeSize float32
}

func (atlas *fontAtlas) mergeFontAwesome(size float32, adjust float32) error {
	fnts := imgui.CurrentIO().Fonts()

	// config for font loading. merging with default font and adjusting offset
	// so that the icons align better.
	mergeConfig := imgui.NewFontConfig()
	defer mergeConfig.Delete()
	mergeConfig.SetMergeMode(true)
	mergeConfig.SetPixelSnapH(true)
	mergeConfig.SetGlyphOffsetY(adjust)

	// limit what glyphs we load
	var glyphBuilder imgui.GlyphRangesBuilder
	glyphBuilder.Add(fonts.FontAwesomeMin, fonts.FontAwesomeMax)

	// merge font awesome
	merge := fnts.AddFontFromMemoryTTFV(fonts.FontAwesome, size, mergeConfig, glyphBuilder.Build().GlyphRanges)
	if merge == 0 {
		return fmt.Errorf("font: error loading font-awesome from memory")
	}

	return nil
}

func (atlas *fontAtlas) initialise(renderer renderer, prefs *preferences) error {
	fnts := imgui.CurrentIO().Fonts()

	err := atlas.setDefaultFont(prefs)
	if err != nil {
		return err
	}

	// load large font awesome
	if atlas.largeFontAwesome == 0 {
		largeFontAwesomeConfig := imgui.NewFontConfig()
		defer largeFontAwesomeConfig.Delete()
		largeFontAwesomeConfig.SetPixelSnapH(true)

		var largeFontAwesomeBuilder imgui.GlyphRangesBuilder
		largeFontAwesomeBuilder.Add(fonts.FontAwesomeMin, fonts.FontAwesomeMax)

		atlas.largeFontAwesomeSize = 22.0
		atlas.largeFontAwesome = fnts.AddFontFromMemoryTTFV(fonts.FontAwesome, atlas.largeFontAwesomeSize, largeFontAwesomeConfig, largeFontAwesomeBuilder.Build().GlyphRanges)
		if atlas.largeFontAwesome == 0 {
			return fmt.Errorf("font: error loading font-awesome from memory")
		}
	}

	// load very-large font awesome
	if atlas.veryLargeFontAwesome == 0 {
		veryLargeFontAwesomeConfig := imgui.NewFontConfig()
		defer veryLargeFontAwesomeConfig.Delete()
		veryLargeFontAwesomeConfig.SetPixelSnapH(true)

		var veryLargeFontAwesomeBuilder imgui.GlyphRangesBuilder
		veryLargeFontAwesomeBuilder.Add(fonts.FontAwesomeMin, fonts.FontAwesomeMax)

		atlas.veryLargeFontAwesomeSize = 44.0
		atlas.veryLargeFontAwesome = fnts.AddFontFromMemoryTTFV(fonts.FontAwesome, atlas.veryLargeFontAwesomeSize, veryLargeFontAwesomeConfig, veryLargeFontAwesomeBuilder.Build().GlyphRanges)
		if atlas.veryLargeFontAwesome == 0 {
			return fmt.Errorf("font: error loading font-awesome from memory")
		}
	}

	// load gopher icons
	if atlas.gopher2600Icons == 0 {
		gopher2600IconConfig := imgui.NewFontConfig()
		defer gopher2600IconConfig.Delete()
		gopher2600IconConfig.SetPixelSnapH(true)
		gopher2600IconConfig.SetGlyphOffsetY(1.0)

		var gopher2600IconBuilder imgui.GlyphRangesBuilder
		gopher2600IconBuilder.Add(fonts.Gopher2600IconMin, fonts.Gopher2600IconMax)

		atlas.gopher2600IconsSize = 60.0
		atlas.gopher2600Icons = fnts.AddFontFromMemoryTTFV(fonts.Gopher2600Icons, atlas.gopher2600IconsSize, gopher2600IconConfig, gopher2600IconBuilder.Build().GlyphRanges)
		if atlas.gopher2600Icons == 0 {
			return fmt.Errorf("font: error loading Gopher2600 font from memory")
		}
	}

	// load diagram font
	if atlas.diagram == 0 {
		diagramConfig := imgui.NewFontConfig()
		defer diagramConfig.Delete()
		diagramConfig.SetPixelSnapH(true)

		var diagramBuilder imgui.GlyphRangesBuilder
		diagramBuilder.Add(fonts.HackMin, fonts.HackMax)

		if atlas.isFreeType() {
			atlas.diagramSize = 10.0
		} else {
			atlas.diagramSize = 11.0
		}
		atlas.diagram = fnts.AddFontFromMemoryTTFV(fonts.Hack, atlas.diagramSize, diagramConfig, diagramBuilder.Build().GlyphRanges)
		if atlas.diagram == 0 {
			return fmt.Errorf("font: error loading hack font from memory")
		}
	}

	// load terminal font
	err = atlas.terminalFont(prefs)
	if err != nil {
		return fmt.Errorf("font: %w", err)
	}

	// load source code font
	err = atlas.sourceCodeFont(prefs)
	if err != nil {
		return fmt.Errorf("font: %w", err)
	}

	// create textures and register with imgui
	tex := renderer.addFontTexture(fnts)
	fnts.SetTextureID(imgui.TextureID(tex.getID()))

	return nil
}
