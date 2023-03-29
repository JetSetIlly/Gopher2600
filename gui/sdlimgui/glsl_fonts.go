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

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui/fonts"
)

type glslFonts struct {
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

	// source code
	code     imgui.Font
	codeSize float32

	// texture used for presentation
	fontTexture uint32
}

func (fnts *glslFonts) destroy() {
	if fnts.fontTexture != 0 {
		gl.DeleteTextures(1, &fnts.fontTexture)
		imgui.CurrentIO().Fonts().SetTextureID(0)
		fnts.fontTexture = 0
	}
}

func (fnts *glslFonts) mergeFontAwesome(size float32, adjust float32) error {
	atlas := imgui.CurrentIO().Fonts()

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
	merge := atlas.AddFontFromMemoryTTFV(fonts.FontAwesome, fnts.defaultFontSize, mergeConfig, glyphBuilder.Build().GlyphRanges)
	if merge == 0 {
		return fmt.Errorf("font: error loading font-awesome from memory")
	}

	return nil
}

func (rnd *glsl) setupFonts() error {
	// only create glslFonts if it doesn't already exist. we'll only load
	// the fonts that we need to. if nothing has changed then all we've
	// done is recreate the font texture
	if rnd.fonts != nil {
		rnd.fonts.destroy()
	} else {
		rnd.fonts = &glslFonts{}
	}

	atlas := imgui.CurrentIO().Fonts()

	err := rnd.fonts.setDefaultFont(rnd.img.prefs)
	if err != nil {
		return err
	}

	// load large font awesome
	if rnd.fonts.largeFontAwesome == 0 {
		largeFontAwesomeConfig := imgui.NewFontConfig()
		defer largeFontAwesomeConfig.Delete()
		largeFontAwesomeConfig.SetPixelSnapH(true)

		var largeFontAwesomeBuilder imgui.GlyphRangesBuilder
		largeFontAwesomeBuilder.Add(fonts.FontAwesomeMin, fonts.FontAwesomeMax)

		rnd.fonts.largeFontAwesomeSize = 22.0
		rnd.fonts.largeFontAwesome = atlas.AddFontFromMemoryTTFV(fonts.FontAwesome, rnd.fonts.largeFontAwesomeSize, largeFontAwesomeConfig, largeFontAwesomeBuilder.Build().GlyphRanges)
		if rnd.fonts.largeFontAwesome == 0 {
			return fmt.Errorf("font: error loading font-awesome from memory")
		}
	}

	// load very-large font awesome
	if rnd.fonts.veryLargeFontAwesome == 0 {
		veryLargeFontAwesomeConfig := imgui.NewFontConfig()
		defer veryLargeFontAwesomeConfig.Delete()
		veryLargeFontAwesomeConfig.SetPixelSnapH(true)

		var veryLargeFontAwesomeBuilder imgui.GlyphRangesBuilder
		veryLargeFontAwesomeBuilder.Add(fonts.FontAwesomeMin, fonts.FontAwesomeMax)

		rnd.fonts.veryLargeFontAwesomeSize = 44.0
		rnd.fonts.veryLargeFontAwesome = atlas.AddFontFromMemoryTTFV(fonts.FontAwesome, rnd.fonts.veryLargeFontAwesomeSize, veryLargeFontAwesomeConfig, veryLargeFontAwesomeBuilder.Build().GlyphRanges)
		if rnd.fonts.veryLargeFontAwesome == 0 {
			return fmt.Errorf("font: error loading font-awesome from memory")
		}
	}

	// load gopher icons
	if rnd.fonts.gopher2600Icons == 0 {
		gopher2600IconConfig := imgui.NewFontConfig()
		defer gopher2600IconConfig.Delete()
		gopher2600IconConfig.SetPixelSnapH(true)
		gopher2600IconConfig.SetGlyphOffsetY(1.0)

		var gopher2600IconBuilder imgui.GlyphRangesBuilder
		gopher2600IconBuilder.Add(fonts.Gopher2600IconMin, fonts.Gopher2600IconMax)

		rnd.fonts.gopher2600IconsSize = 60.0
		rnd.fonts.gopher2600Icons = atlas.AddFontFromMemoryTTFV(fonts.Gopher2600Icons, rnd.fonts.gopher2600IconsSize, gopher2600IconConfig, gopher2600IconBuilder.Build().GlyphRanges)
		if rnd.fonts.gopher2600Icons == 0 {
			return fmt.Errorf("font: error loading Gopher2600 font from memory")
		}
	}

	// load diagram font
	if rnd.fonts.diagram == 0 {
		diagramConfig := imgui.NewFontConfig()
		defer diagramConfig.Delete()
		diagramConfig.SetPixelSnapH(true)

		var diagramBuilder imgui.GlyphRangesBuilder
		diagramBuilder.Add(fonts.HackMin, fonts.HackMax)

		if rnd.fonts.isFreeType() {
			rnd.fonts.diagramSize = 10.0
		} else {
			rnd.fonts.diagramSize = 11.0
		}
		rnd.fonts.diagram = atlas.AddFontFromMemoryTTFV(fonts.Hack, rnd.fonts.diagramSize, diagramConfig, diagramBuilder.Build().GlyphRanges)
		if rnd.fonts.diagram == 0 {
			return fmt.Errorf("font: error loading hack font from memory")
		}
	}

	// load source code font
	err = rnd.fonts.sourceCodeFont(rnd.img.prefs)
	if err != nil {
		return fmt.Errorf("font: %w", err)
	}

	// create font texture
	image := atlas.TextureDataAlpha8()
	gl.GenTextures(1, &rnd.fonts.fontTexture)
	gl.BindTexture(gl.TEXTURE_2D, rnd.fonts.fontTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, int32(image.Width), int32(image.Height), 0, gl.RED, gl.UNSIGNED_BYTE, image.Pixels)
	atlas.SetTextureID(imgui.TextureID(rnd.fonts.fontTexture))

	return nil
}
