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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlimgui

import (
	"gopher2600/gui"
	"gopher2600/gui/sdlaudio"
	"gopher2600/paths"
	"gopher2600/performance/limiter"
	"gopher2600/television"
	"gopher2600/test"
	"io"

	"github.com/inkyblackness/imgui-go/v2"
)

// SdlImgui is an sdl based visualiser using imgui
type SdlImgui struct {
	io      imgui.IO
	context *imgui.Context
	plt     *platform
	audio   *sdlaudio.Audio
	glsl    *glsl

	tv     television.Television
	screen *tvScreen

	// functions that need to be performed in the main thread should be queued
	// for service.
	service    chan func()
	serviceErr chan error

	// limit number of frames per second
	lmtr *limiter.FpsLimiter

	// events channel is not created but assigned with SetEventChannel()
	events chan gui.Event

	// window opening is delayed until television frame is stable
	showOnNextStable bool

	// mouse coords at last frame
	mx, my int32

	// whether mouse is captured
	isCaptured bool
}

// NewSdlImgui is the preferred method of initialisation for type SdlWindows
//
// MUST ONLY be called from the #mainthread
func NewSdlImgui(tv television.Television) (*SdlImgui, error) {
	test.AssertMainThread()

	img := &SdlImgui{
		context:    imgui.CreateContext(nil),
		io:         imgui.CurrentIO(),
		tv:         tv,
		service:    make(chan func(), 1),
		serviceErr: make(chan error, 1),
	}

	// create new frame limiter. we change the rate in the resize function
	// (rate may change due to specification change)
	img.lmtr = limiter.NewFPSLimiter(-1)

	var err error

	img.plt, err = newPlatform(img)
	if err != nil {
		return nil, err
	}

	img.glsl, err = newGlsl(img.io)
	if err != nil {
		return nil, err
	}

	iniPath, err := paths.ResourcePath("", "imgui.ini")
	if err != nil {
		return nil, err
	}
	img.io.SetIniFilename(iniPath)

	img.screen, err = newTvScreen(img)
	if err != nil {
		return nil, err
	}
	img.glsl.tvScreenTexture = img.screen.texture

	tv.AddPixelRenderer(img.screen)

	img.audio, err = sdlaudio.NewAudio()
	if err != nil {
		return nil, err
	}
	tv.AddAudioMixer(img.audio)

	return img, nil
}

// Destroy implements GuiCreator interface
//
// MUST ONLY be called from the #mainthread
func (img *SdlImgui) Destroy(output io.Writer) {
	test.AssertMainThread()

	img.audio.EndMixing()
	img.screen.destroy()
	img.glsl.destroy()
	img.plt.destroy()
	img.context.Destroy()
}

// SetEventChannel implements gui.GUI interface
//
// MUST ONLY be called from the #mainthread
func (img *SdlImgui) SetEventChannel(events chan gui.Event) {
	test.AssertMainThread()
	img.events = events
}
