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
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/imgui-go/v5"
)

type fontDisplay interface {
	displayDPI() (float32, error)
	windowSize() (width, height float32)
}

type fontAtlas struct {
	// gui font
	gui     imgui.Font
	guiSize float32

	smallerGui     imgui.Font
	smallerGuiSize float32

	tinyGui     imgui.Font
	tinyGuiSize float32

	// used for notifications (eg. network access, etc.)
	largeFontAwesome     imgui.Font
	largeFontAwesomeSize float32

	// used for rewind/fast-forward state
	veryLargeFontAwesome     imgui.Font
	veryLargeFontAwesomeSize float32

	// custom icons for controllers and other peripherals
	gopher2600Icons          imgui.Font
	gopher2600IconsSize      float32
	smallGopher2600Icons     imgui.Font
	smallGopher2600IconsSize float32

	// annotation of diagrams
	diagram     imgui.Font
	diagramSize float32

	// terminal
	terminal     imgui.Font
	terminalSize float32

	// source code
	code     imgui.Font
	codeSize float32

	// subtitles are a little different to other fonts. 10 different sizes are preloaded and then
	// when the window is resized, the preloaded size nearest to the required size is used
	subtitles     [10]imgui.Font
	subtitlesSize [10]float32
	subtitlesIdx  int
}

func scaleFontForDPI(pt float32, dpi float32) float32 {
	return float32(math.Round(float64(pt * (dpi / 72))))
}

type fontSpec struct {
	fonts.FontSpec
	size  float32
	cfg   imgui.FontConfig
	merge bool
}

func (atlas *fontAtlas) loadFont(spec fontSpec) (imgui.Font, error) {
	cfg := spec.cfg
	if cfg == 0 {
		cfg = imgui.NewFontConfig()
		defer cfg.Delete()
	}

	var builder imgui.GlyphRangesBuilder
	builder.Add(spec.Min, spec.Max)

	f := imgui.CurrentIO().Fonts().AddFontFromMemoryTTFV(spec.Data, spec.size, cfg, builder.Build().GlyphRanges)
	if f == 0 {
		return 0, fmt.Errorf("error loading font from memory")
	}

	if spec.merge {
		atlas.mergeFontAwesome(spec.size, 1.0)
	}
	return f, nil
}

func (atlas *fontAtlas) mergeFontAwesome(size float32, adjust float32) error {
	cfg := imgui.NewFontConfig()
	defer cfg.Delete()
	cfg.SetMergeMode(true)
	cfg.SetGlyphOffsetY(adjust)

	// limit what glyphs we load
	var glyphBuilder imgui.GlyphRangesBuilder
	glyphBuilder.Add(fonts.FontAwesome.Min, fonts.FontAwesome.Max)

	// merge font awesome
	merge := imgui.CurrentIO().Fonts().AddFontFromMemoryTTFV(fonts.FontAwesome.Data, size, cfg, glyphBuilder.Build().GlyphRanges)
	if merge == 0 {
		return fmt.Errorf("font: error loading font-awesome from memory")
	}

	return nil
}

func (atlas *fontAtlas) loadFonts(display fontDisplay, renderer renderer, prefs *preferences) error {
	setFontBuilderFlags(imgui.CurrentIO().Fonts())
	dpi, err := display.displayDPI()
	if err != nil {
		return fmt.Errorf("font: %w", err)
	}
	logger.Logf(logger.Allow, "fonts", "using dpi value of %v", dpi)

	// the first font we load will be the default font
	sz := float32(prefs.guiFontSize.Get().(int))
	sz = scaleFontForDPI(sz, dpi)
	if sz != atlas.guiSize {
		var err error

		atlas.guiSize = sz
		atlas.gui, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.JetBrainsMono,
			size:     atlas.guiSize,
			merge:    true,
		})
		if err != nil {
			return fmt.Errorf("gui font: %w", err)
		}
		logger.Logf(logger.Allow, "fonts", "gui font size of %v", atlas.guiSize)

		// load small gui font based on the size of the normal gui font
		atlas.smallerGuiSize = atlas.guiSize * 0.95
		atlas.smallerGui, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.JetBrainsMono,
			size:     atlas.smallerGuiSize,
			merge:    true,
		})
		if err != nil {
			return fmt.Errorf("smaller gui font: %w", err)
		}
		logger.Logf(logger.Allow, "fonts", "smaller gui font size of %v", atlas.smallerGuiSize)

		// load tiny small gui font based on the size of the normal gui font
		atlas.tinyGuiSize = atlas.guiSize * 0.87
		atlas.tinyGui, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.JetBrainsMono,
			size:     atlas.tinyGuiSize,
			merge:    true,
		})
		if err != nil {
			return fmt.Errorf("tiny gui font: %w", err)
		}
		logger.Logf(logger.Allow, "fonts", "tiny gui font size of %v", atlas.tinyGuiSize)
	}

	// load terminal font
	sz = float32(prefs.terminalFontSize.Get().(int))
	sz = scaleFontForDPI(sz, dpi)
	if sz != atlas.terminalSize {
		atlas.terminalSize = sz

		var err error
		atlas.terminal, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.JetBrainsMono,
			size:     atlas.terminalSize,
			merge:    true,
		})
		if err != nil {
			return fmt.Errorf("terminal font: %w", err)
		}
		logger.Logf(logger.Allow, "fonts", "terminal font size of %v", atlas.terminalSize)
	}

	// load code font
	sz = float32(prefs.codeFontSize.Get().(int))
	sz = scaleFontForDPI(sz, dpi)
	if sz != atlas.codeSize {
		atlas.codeSize = sz

		var err error
		atlas.code, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.JetBrainsMono,
			size:     atlas.codeSize,
			merge:    true,
		})
		if err != nil {
			return fmt.Errorf("code font: %w", err)
		}
		logger.Logf(logger.Allow, "fonts", "code font size of %v", atlas.codeSize)
	}

	// load a range of subtitle sizes
	var subtitlesChanged bool
	for i := range len(atlas.subtitlesSize) {
		const baseSubtitleSize = 10
		sz := float32(baseSubtitleSize * (i + 1))
		sz = scaleFontForDPI(sz, dpi)
		if sz != atlas.subtitlesSize[i] {
			subtitlesChanged = true
			atlas.subtitlesSize[i] = sz
			atlas.subtitles[i], err = atlas.loadFont(fontSpec{
				FontSpec: fonts.JetBrainsMonoBold_ReducedRange,
				size:     atlas.subtitlesSize[i],
			})
			if err != nil {
				return fmt.Errorf("subtitle font: %w", err)
			}
		}
	}
	if subtitlesChanged {
		logger.Logf(logger.Allow, "fonts", "subtitle font sizes of %v to %v",
			atlas.subtitlesSize[0],
			atlas.subtitlesSize[len(atlas.subtitlesSize)-1])
	}

	// the remaining fonts require finessing with config changes
	// altering SetPixelSnapH() and SetGlyphOffsetY()
	cfg := imgui.NewFontConfig()
	defer cfg.Delete()

	// load diagram font
	sz = atlas.guiSize * 0.75
	if sz != atlas.diagramSize {
		cfg.SetPixelSnapH(true)
		cfg.SetGlyphOffsetY(0.0)
		atlas.diagramSize = sz
		atlas.diagram, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.Hack,
			size:     atlas.diagramSize,
			cfg:      cfg,
			merge:    true,
		})
		logger.Logf(logger.Allow, "fonts", "diagram font size of %v", atlas.diagramSize)
		if err != nil {
			return fmt.Errorf("diagram font: %w", err)
		}
	}

	// load large font awesome icons
	sz = 22.0 // not DPI adjusted
	if sz != atlas.largeFontAwesomeSize {
		cfg.SetPixelSnapH(true)
		cfg.SetGlyphOffsetY(0.0)
		atlas.largeFontAwesomeSize = sz
		atlas.largeFontAwesome, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.FontAwesome,
			size:     atlas.largeFontAwesomeSize,
			cfg:      cfg,
		})
		if err != nil {
			return fmt.Errorf("large font awesome: %w", err)
		}
		logger.Logf(logger.Allow, "fonts", "large icon size of %v", atlas.largeFontAwesomeSize)
	}

	// load very-large font awesome icons
	sz = 44.0 // not DPI adjusted
	if sz != atlas.veryLargeFontAwesomeSize {
		cfg.SetPixelSnapH(true)
		atlas.veryLargeFontAwesomeSize = sz
		atlas.veryLargeFontAwesome, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.FontAwesome,
			size:     atlas.veryLargeFontAwesomeSize,
			cfg:      cfg,
		})
		if err != nil {
			return fmt.Errorf("very large font awesome: %w", err)
		}
		logger.Logf(logger.Allow, "fonts", "very large icon size of %v", atlas.veryLargeFontAwesomeSize)
	}

	// load gopher icons
	sz = 60.0 // not DPI adjusted
	if sz != atlas.gopher2600IconsSize {
		cfg.SetPixelSnapH(true)
		cfg.SetGlyphOffsetY(1.0)
		atlas.gopher2600IconsSize = sz
		atlas.gopher2600Icons, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.Gopher2600Icons,
			size:     atlas.gopher2600IconsSize,
			cfg:      cfg,
		})
		if err != nil {
			return fmt.Errorf("gopher2600 icons font: %w", err)
		}
		logger.Logf(logger.Allow, "fonts", "gopher2600 icon size of %v", atlas.gopher2600IconsSize)
	}

	// load small gopher icons
	sz = 30.0 // not DPI adjusted
	if sz != atlas.smallGopher2600IconsSize {
		cfg.SetPixelSnapH(true)
		cfg.SetGlyphOffsetY(1.0)
		atlas.smallGopher2600IconsSize = sz
		atlas.smallGopher2600Icons, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.Gopher2600Icons,
			size:     atlas.smallGopher2600IconsSize,
			cfg:      cfg,
		})
		logger.Logf(logger.Allow, "fonts", "small gopher2600 icon size of %v", atlas.smallGopher2600IconsSize)
		if err != nil {
			return fmt.Errorf("small gopher2600 icons font: %w", err)
		}
	}

	// create textures and register with imgui
	atlas.resize(display)
	tex := renderer.addFontTexture(imgui.CurrentIO().Fonts())
	imgui.CurrentIO().Fonts().SetTextureID(imgui.TextureID(tex.getID()))

	return nil
}

// resize is called when the containing window is resized
func (atlas *fontAtlas) resize(display fontDisplay) error {
	dpi, err := display.displayDPI()
	if err != nil {
		return fmt.Errorf("font: %w", err)
	}
	_, winh := display.windowSize()
	sz := float32(winh * 0.05)
	sz = scaleFontForDPI(sz, dpi)
	for i := range len(atlas.subtitlesSize) {
		if sz >= atlas.subtitlesSize[i] {
			atlas.subtitlesIdx = i
		}
	}
	return nil
}
