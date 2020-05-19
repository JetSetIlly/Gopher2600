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

package sdlimgui_play

import (
	"io"

	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/gui/sdlaudio"
	"github.com/jetsetilly/gopher2600/television"

	"github.com/inkyblackness/imgui-go/v2"
)

const imguiIniFile = "playmode_imgui.ini"

// SdlImgui is an sdl based visualiser using imgui
type SdlImguiPlay struct {
	// the mechanical requirements for the gui
	io      imgui.IO
	context *imgui.Context
	plt     *platform
	glsl    *glsl

	// references to the emulation
	tv television.Television

	// implementations of screen television protocols
	screen *screen
	audio  *sdlaudio.Audio

	scr *winScreen

	// functions that need to be performed in the main thread should be queued
	// for service
	service    chan func()
	serviceErr chan error

	// SetFeature() hands off requests to the featureReq channel for servicing.
	// think of this as a special instance of the service chan
	featureReq chan featureRequest
	featureErr chan error

	// events channel is not created but assigned with SetEventChannel()
	events chan gui.Event

	// mouse coords at last frame
	mx, my int32
}

// NewSdlImguiPlay is the preferred method of initialisation for type SdlImguiPlay
//
// MUST ONLY be called from the #mainthread
func NewSdlImguiPlay(tv television.Television) (*SdlImguiPlay, error) {
	img := &SdlImguiPlay{
		context:    imgui.CreateContext(nil),
		io:         imgui.CurrentIO(),
		tv:         tv,
		service:    make(chan func(), 1),
		serviceErr: make(chan error, 1),
		featureReq: make(chan featureRequest, 1),
		featureErr: make(chan error, 1),
	}

	var err error

	img.plt, err = newPlatform(img)
	if err != nil {
		return nil, err
	}

	img.glsl, err = newGlsl(img.io, img)
	if err != nil {
		return nil, err
	}

	// we don't want to load or save an imgui ini file
	img.io.SetIniFilename("")

	img.screen = newScreen(img)
	img.scr, err = newWinScreen(img)
	if err != nil {
		return nil, err
	}

	// connect some screen properties to other parts of the system
	img.glsl.screenTexture = img.screen.screenTexture
	tv.AddPixelRenderer(img.screen)

	// this audio mixer produces the sound. there is another AudioMixer
	// implementation in winAudio which visualises the sound
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
func (img *SdlImguiPlay) Destroy(output io.Writer) {
	img.audio.EndMixing()
	img.glsl.destroy()

	err := img.plt.destroy()
	if err != nil {
		output.Write([]byte(err.Error()))
	}

	img.context.Destroy()
}

func (img *SdlImguiPlay) draw() {
	img.scr.draw()
}
