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
	windowSize() (width, height float32)
}

type fontAtlas struct {
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

	// the first font we load will be the default font
	sz := float32(prefs.guiFontSize.Get().(int))
	sz = scaleFontForDPI(sz, dpi)
	if sz != atlas.guiSize {
		atlas.guiSize = sz
		var err error
		atlas.gui, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.JetBrainsMono,
			size:     atlas.guiSize,
			merge:    true,
		})

		if err != nil {
			return fmt.Errorf("gui font: %w", err)
		}

		// load small gui font based on the size of the normal gui font
		atlas.smallGuiSize = sz * 0.85
		atlas.smallGuiSize = scaleFontForDPI(atlas.smallGuiSize, dpi)

		atlas.smallGui, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.JetBrainsMono,
			size:     atlas.smallGuiSize,
			merge:    true,
		})

		if err != nil {
			return fmt.Errorf("gui font: %w", err)
		}
	}

	// load large font awesome
	if atlas.largeFontAwesome == 0 {
		cfg := imgui.NewFontConfig()
		defer cfg.Delete()
		cfg.SetPixelSnapH(true)

		atlas.largeFontAwesomeSize = 22.0

		var err error
		atlas.largeFontAwesome, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.FontAwesome,
			size:     atlas.largeFontAwesomeSize,
			cfg:      cfg,
		})

		if err != nil {
			return fmt.Errorf("large font awesome: %w", err)
		}
	}

	// load very-large font awesome
	if atlas.veryLargeFontAwesome == 0 {
		cfg := imgui.NewFontConfig()
		defer cfg.Delete()
		cfg.SetPixelSnapH(true)

		atlas.veryLargeFontAwesomeSize = 44.0

		var err error
		atlas.veryLargeFontAwesome, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.FontAwesome,
			size:     atlas.veryLargeFontAwesomeSize,
			cfg:      cfg,
		})

		if err != nil {
			return fmt.Errorf("very large font awesome: %w", err)
		}
	}

	// load gopher icons
	if atlas.gopher2600Icons == 0 {
		cfg := imgui.NewFontConfig()
		defer cfg.Delete()
		cfg.SetPixelSnapH(true)
		cfg.SetGlyphOffsetY(1.0)

		atlas.gopher2600IconsSize = 60.0

		var err error
		atlas.gopher2600Icons, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.Gopher2600Icons,
			size:     atlas.gopher2600IconsSize,
			cfg:      cfg,
		})

		if err != nil {
			return fmt.Errorf("gopher2600 icons font: %w", err)
		}
	}

	// load diagram font
	if atlas.diagram == 0 {
		cfg := imgui.NewFontConfig()
		defer cfg.Delete()
		cfg.SetPixelSnapH(true)

		atlas.diagramSize = scaleFontForDPI(10.0, dpi)

		var err error
		atlas.diagram, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.Hack,
			size:     atlas.diagramSize,
			cfg:      cfg,
			merge:    true,
		})

		if err != nil {
			return fmt.Errorf("diagram font: %w", err)
		}
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
			return fmt.Errorf("code font: %w", err)
		}
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
	}

	// load subtitle font at 5% of window size
	_, h := display.windowSize()
	sz = float32(h * 0.05)
	sz = scaleFontForDPI(sz, dpi)
	if sz != atlas.subtitlesSize {
		atlas.subtitlesSize = sz

		atlas.subtitles, err = atlas.loadFont(fontSpec{
			FontSpec: fonts.JetBrainsMonoBold_ReducedRange,
			size:     atlas.subtitlesSize,
		})

		if err != nil {
			return fmt.Errorf("code font: %w", err)
		}
	}

	// create textures and register with imgui
	tex := renderer.addFontTexture(imgui.CurrentIO().Fonts())
	imgui.CurrentIO().Fonts().SetTextureID(imgui.TextureID(tex.getID()))

	return nil
}
