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
	"io"

	"github.com/jetsetilly/imgui-go/v5"
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
	isScreenshotting() bool
	record(enable bool, w io.Writer, lastFrame int) error
	isRecording() bool
	addTexture(typ shaderType, linear bool, clamp bool, config any) texture
	addFontTexture(fnt imgui.FontAtlas) texture
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
)

type texture interface {
	getID() uint32
	markForCreation()
	clear()
	render(*image.RGBA)
}

type screenshotMode string

const (
	modeSingle   screenshotMode = "single"
	modeDouble   screenshotMode = "double"
	modeTriple   screenshotMode = "triple"
	modeMovement screenshotMode = "movement"
)

type screenshotResult struct {
	// a description of the screenshot as provided by the renderer
	description string

	// the final image
	image *image.RGBA

	// any errors that were encountered in the screenshotting preperation
	err error
}
