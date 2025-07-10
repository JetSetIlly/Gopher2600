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
	"math"

	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/imgui-go/v5"
)

type fontDisplay interface {
	displayDPI() (float32, error)
}

type fontAtlas struct {
	display fontDisplay

	// gui font
	gui     imgui.Font
	guiSize float32

	smallGui     imgui.Font
	smallGuiSize float32

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

	// atarivox subtitles
	subtitles     imgui.Font
	subtitlesSize float32
}

func scaleFontForDPI(pt float32, dpi float32) float32 {
	return float32(math.Round(float64(pt * (dpi / 72))))
}

func (atlas *fontAtlas) initialise(display fontDisplay, renderer renderer, prefs *preferences) error {
	setFontBuilderFlags(imgui.CurrentIO().Fonts())
	dpi, err := display.displayDPI()
	if err != nil {
		return fmt.Errorf("font: %w", err)
	}

	err = atlas.setDefaultFont(prefs, dpi)
	if err != nil {
		return err
	}

	// load small gui font
	if atlas.smallGui == 0 {
		smallGuiConfig := imgui.NewFontConfig()
		defer smallGuiConfig.Delete()
		smallGuiConfig.SetPixelSnapH(true)

		var smallGuiBuilder imgui.GlyphRangesBuilder
		smallGuiBuilder.Add(fonts.JetBrainsMonoMin, fonts.JetBrainsMonoMax)

		atlas.smallGuiSize = float32(prefs.guiFontSize.Get().(int)) * 0.85
		atlas.smallGuiSize = scaleFontForDPI(atlas.smallGuiSize, dpi)
		atlas.smallGui = imgui.CurrentIO().Fonts().AddFontFromMemoryTTFV(fonts.JetBrainsMono, atlas.smallGuiSize, smallGuiConfig, smallGuiBuilder.Build().GlyphRanges)
		if atlas.smallGui == 0 {
			return fmt.Errorf("font: error loading JetBrainsMono font from memory")
		}
	}

	// load large font awesome
	if atlas.largeFontAwesome == 0 {
		largeFontAwesomeConfig := imgui.NewFontConfig()
		defer largeFontAwesomeConfig.Delete()
		largeFontAwesomeConfig.SetPixelSnapH(true)

		var largeFontAwesomeBuilder imgui.GlyphRangesBuilder
		largeFontAwesomeBuilder.Add(fonts.FontAwesomeMin, fonts.FontAwesomeMax)

		atlas.largeFontAwesomeSize = 22.0
		atlas.largeFontAwesome = imgui.CurrentIO().Fonts().AddFontFromMemoryTTFV(fonts.FontAwesome, atlas.largeFontAwesomeSize, largeFontAwesomeConfig, largeFontAwesomeBuilder.Build().GlyphRanges)
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
		atlas.veryLargeFontAwesome = imgui.CurrentIO().Fonts().AddFontFromMemoryTTFV(fonts.FontAwesome, atlas.veryLargeFontAwesomeSize, veryLargeFontAwesomeConfig, veryLargeFontAwesomeBuilder.Build().GlyphRanges)
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
		atlas.gopher2600Icons = imgui.CurrentIO().Fonts().AddFontFromMemoryTTFV(fonts.Gopher2600Icons, atlas.gopher2600IconsSize, gopher2600IconConfig, gopher2600IconBuilder.Build().GlyphRanges)
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

		atlas.diagramSize = scaleFontForDPI(10.0, dpi)
		atlas.diagram = imgui.CurrentIO().Fonts().AddFontFromMemoryTTFV(fonts.Hack, atlas.diagramSize, diagramConfig, diagramBuilder.Build().GlyphRanges)
		if atlas.diagram == 0 {
			return fmt.Errorf("font: error loading hack font from memory")
		}
	}

	// load terminal font
	err = atlas.terminalFont(prefs, dpi)
	if err != nil {
		return fmt.Errorf("font: %w", err)
	}

	// load source code font
	err = atlas.sourceCodeFont(prefs, dpi)
	if err != nil {
		return fmt.Errorf("font: %w", err)
	}

	// load source code font
	err = atlas.subtitlesFont(prefs, dpi)
	if err != nil {
		return fmt.Errorf("font: %w", err)
	}

	// create textures and register with imgui
	tex := renderer.addFontTexture(imgui.CurrentIO().Fonts())
	imgui.CurrentIO().Fonts().SetTextureID(imgui.TextureID(tex.getID()))

	return nil
}

func (atlas *fontAtlas) setDefaultFont(prefs *preferences, dpi float32) error {
	guiFontSize := float32(prefs.guiFontSize.Get().(int))
	if atlas.guiSize != 0 && guiFontSize == atlas.guiSize {
		return nil
	}
	atlas.guiSize = guiFontSize

	// load gui font
	cfg := imgui.NewFontConfig()
	defer cfg.Delete()

	var builder imgui.GlyphRangesBuilder
	builder.Add(fonts.JetBrainsMonoMin, fonts.JetBrainsMonoMax)

	guiFontSize = scaleFontForDPI(guiFontSize, dpi)
	atlas.gui = imgui.CurrentIO().Fonts().AddFontFromMemoryTTFV(fonts.JetBrainsMono, guiFontSize, cfg, builder.Build().GlyphRanges)
	if atlas.gui == 0 {
		return fmt.Errorf("font: error loading JetBrainsMono font from memory")
	}

	atlas.mergeFontAwesome(guiFontSize, 1.0)

	return nil
}

func (atlas *fontAtlas) sourceCodeFont(prefs *preferences, dpi float32) error {
	codeSize := float32(prefs.codeFontSize.Get().(int))
	if atlas.codeSize != 0 && codeSize == atlas.codeSize {
		return nil
	}
	atlas.codeSize = codeSize

	cfg := imgui.NewFontConfig()
	defer cfg.Delete()

	var builder imgui.GlyphRangesBuilder
	builder.Add(fonts.JetBrainsMonoMin, fonts.JetBrainsMonoMax)

	codeSize = scaleFontForDPI(codeSize, dpi)
	atlas.code = imgui.CurrentIO().Fonts().AddFontFromMemoryTTFV(fonts.JetBrainsMono, codeSize, cfg, builder.Build().GlyphRanges)
	if atlas.code == 0 {
		return fmt.Errorf("font: error loading JetBrainsMono font from memory")
	}

	atlas.mergeFontAwesome(codeSize, 0.0)

	return nil
}

func (atlas *fontAtlas) terminalFont(prefs *preferences, dpi float32) error {
	terminalSize := float32(prefs.terminalFontSize.Get().(int))
	if atlas.terminalSize != 0 && terminalSize == atlas.terminalSize {
		return nil
	}
	atlas.terminalSize = terminalSize

	cfg := imgui.NewFontConfig()
	defer cfg.Delete()

	var builder imgui.GlyphRangesBuilder
	builder.Add(fonts.JetBrainsMonoMin, fonts.JetBrainsMonoMax)

	terminalSize = scaleFontForDPI(terminalSize, dpi)
	atlas.terminal = imgui.CurrentIO().Fonts().AddFontFromMemoryTTFV(fonts.JetBrainsMono, terminalSize, cfg, builder.Build().GlyphRanges)
	if atlas.terminal == 0 {
		return fmt.Errorf("font: error loading JetBrainsMono font from memory")
	}

	atlas.mergeFontAwesome(terminalSize, 0.0)

	return nil
}

func (atlas *fontAtlas) subtitlesFont(prefs *preferences, dpi float32) error {
	subtitlesSize := float32(20.0)
	if atlas.subtitlesSize != 0 && subtitlesSize == atlas.subtitlesSize {
		return nil
	}
	atlas.subtitlesSize = subtitlesSize

	cfg := imgui.NewFontConfig()
	defer cfg.Delete()

	var builder imgui.GlyphRangesBuilder
	builder.Add(fonts.SubtitleMin, fonts.SubtitleMax)

	subtitlesSize = scaleFontForDPI(subtitlesSize, dpi)
	atlas.subtitles = imgui.CurrentIO().Fonts().AddFontFromMemoryTTFV(fonts.Subtitle, subtitlesSize, cfg, builder.Build().GlyphRanges)
	if atlas.code == 0 {
		return fmt.Errorf("font: error loading Hack font from memory")
	}

	return nil
}

func (atlas *fontAtlas) mergeFontAwesome(size float32, adjust float32) error {
	mergeConfig := imgui.NewFontConfig()
	defer mergeConfig.Delete()
	mergeConfig.SetMergeMode(true)
	mergeConfig.SetGlyphOffsetY(adjust)

	// limit what glyphs we load
	var glyphBuilder imgui.GlyphRangesBuilder
	glyphBuilder.Add(fonts.FontAwesomeMin, fonts.FontAwesomeMax)

	// merge font awesome
	merge := imgui.CurrentIO().Fonts().AddFontFromMemoryTTFV(fonts.FontAwesome, size, mergeConfig, glyphBuilder.Build().GlyphRanges)
	if merge == 0 {
		return fmt.Errorf("font: error loading font-awesome from memory")
	}

	return nil
}
