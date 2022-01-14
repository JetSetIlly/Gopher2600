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
	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/gui/fonts"
)

type glslFonts struct {
	largeFontAwesome     imgui.Font
	largeFontAwesomeSize float32

	veryLargeFontAwesome     imgui.Font
	veryLargeFontAwesomeSize float32

	gopher2600Icons     imgui.Font
	gopher2600IconsSize float32

	hack     imgui.Font
	hackSize float32

	fontTexture uint32
}

func (fnts *glslFonts) destroy() {
	if fnts.fontTexture != 0 {
		gl.DeleteTextures(1, &fnts.fontTexture)
		imgui.CurrentIO().Fonts().SetTextureID(0)
		fnts.fontTexture = 0
	}
}

func newGLSLfonts() (*glslFonts, error) {
	fnts := &glslFonts{}

	// add default font
	atlas := imgui.CurrentIO().Fonts()
	atlas.AddFontDefault()

	// config for font loading. merging with default font and adjusting offset
	// so that the icons align better.
	mergeConfig := imgui.NewFontConfig()
	defer mergeConfig.Delete()
	mergeConfig.SetMergeMode(true)
	mergeConfig.SetPixelSnapH(true)
	mergeConfig.SetGlyphOffsetY(2.0)

	// limit what glyphs we load
	var glyphBuilder imgui.GlyphRangesBuilder
	glyphBuilder.Add(fonts.FontAwesomeMin, fonts.FontAwesomeMax)

	// load font awesome
	font := atlas.AddFontFromMemoryTTFV(fonts.FontAwesome, 13.0, mergeConfig, glyphBuilder.Build().GlyphRanges)
	if font == 0 {
		return nil, curated.Errorf("font: error loading font from memory")
	}

	// load large icons
	largeFontAwesomeConfig := imgui.NewFontConfig()
	defer largeFontAwesomeConfig.Delete()
	largeFontAwesomeConfig.SetPixelSnapH(true)

	var largeFontAwesomeBuilder imgui.GlyphRangesBuilder
	largeFontAwesomeBuilder.Add(fonts.FontAwesomeMin, fonts.FontAwesomeMax)

	fnts.largeFontAwesomeSize = 22.0
	fnts.largeFontAwesome = atlas.AddFontFromMemoryTTFV(fonts.FontAwesome, fnts.largeFontAwesomeSize, largeFontAwesomeConfig, largeFontAwesomeBuilder.Build().GlyphRanges)
	if font == 0 {
		return nil, curated.Errorf("font: error loading large FA font from memory")
	}

	// load very-large icons
	veryLargeFontAwesomeConfig := imgui.NewFontConfig()
	defer veryLargeFontAwesomeConfig.Delete()
	veryLargeFontAwesomeConfig.SetPixelSnapH(true)

	var veryLargeFontAwesomeBuilder imgui.GlyphRangesBuilder
	veryLargeFontAwesomeBuilder.Add(fonts.FontAwesomeMin, fonts.FontAwesomeMax)

	fnts.veryLargeFontAwesomeSize = 44.0
	fnts.veryLargeFontAwesome = atlas.AddFontFromMemoryTTFV(fonts.FontAwesome, fnts.veryLargeFontAwesomeSize, veryLargeFontAwesomeConfig, veryLargeFontAwesomeBuilder.Build().GlyphRanges)
	if font == 0 {
		return nil, curated.Errorf("font: error loading very large FA font from memory")
	}

	// load gopher icons
	gopher2600IconConfig := imgui.NewFontConfig()
	defer gopher2600IconConfig.Delete()
	gopher2600IconConfig.SetPixelSnapH(true)
	gopher2600IconConfig.SetGlyphOffsetY(1.0)

	var gopher2600IconBuilder imgui.GlyphRangesBuilder
	gopher2600IconBuilder.Add(fonts.Gopher2600IconMin, fonts.Gopher2600IconMax)

	fnts.gopher2600IconsSize = 52.0
	fnts.gopher2600Icons = atlas.AddFontFromMemoryTTFV(fonts.Gopher2600Icons, fnts.gopher2600IconsSize, gopher2600IconConfig, gopher2600IconBuilder.Build().GlyphRanges)
	if font == 0 {
		return nil, curated.Errorf("font: error loading Gopher2600 font from memory")
	}

	// load hack font
	hackConfig := imgui.NewFontConfig()
	defer hackConfig.Delete()
	hackConfig.SetPixelSnapH(true)
	hackConfig.SetGlyphOffsetY(1.0)

	var hackBuilder imgui.GlyphRangesBuilder
	hackBuilder.Add(fonts.HackMin, fonts.HackMax)

	fnts.hackSize = 11.0
	fnts.hack = atlas.AddFontFromMemoryTTFV(fonts.Hack, fnts.hackSize, hackConfig, hackBuilder.Build().GlyphRanges)
	if font == 0 {
		return nil, curated.Errorf("font: error loading hack font from memory")
	}

	// create font texture
	image := atlas.TextureDataAlpha8()
	gl.GenTextures(1, &fnts.fontTexture)
	gl.BindTexture(gl.TEXTURE_2D, fnts.fontTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, int32(image.Width), int32(image.Height), 0, gl.RED, gl.UNSIGNED_BYTE, image.Pixels)
	atlas.SetTextureID(imgui.TextureID(fnts.fontTexture))

	return fnts, nil
}
