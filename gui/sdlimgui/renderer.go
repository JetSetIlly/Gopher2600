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
	"image"

	"github.com/inkyblackness/imgui-go/v4"
)

type requirement int

const (
	requiresOpenGL32 requirement = iota
	requiresOpenGL21
)

type renderer interface {
	requires() requirement
	supportsCRT() bool
	start() error
	destroy()
	preRender()
	render()
	screenshot(mode screenshotMode, finish chan screenshotResult)
	addTexture(typ shaderType, linear bool, clamp bool) texture
	addFontTexture(fnt imgui.FontAtlas) texture
	pushTVColour()
	popTVColour()
}

type shaderType int

const (
	shaderNone shaderType = iota
	shaderGUI
	shaderColor
	shaderPlayscr
	shaderBevel
	shaderDbgScr
	shaderDbgScrOverlay
	shaderTVColour
)

type texture interface {
	getID() uint32
	markForCreation()
	clear()
	render(*image.RGBA)
}

type screenshotMode string

const (
	modeSingle    screenshotMode = "single"
	modeFlicker   screenshotMode = "flicker"
	modeComposite screenshotMode = "composite"
)

type screenshotResult struct {
	// a description of the screenshot as provided by the renderer
	description string

	// the final image
	image *image.RGBA

	// any errors that were encountered in the screenshotting preperation
	err error
}
