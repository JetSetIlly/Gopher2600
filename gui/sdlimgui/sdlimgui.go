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
	"fmt"
	"io"

	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/gui/sdlaudio"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/lazyvalues"
	"github.com/jetsetilly/gopher2600/paths"
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/television"

	"github.com/inkyblackness/imgui-go/v2"
)

const imguiIniFile = "debugger_imgui.ini"

// SdlImgui is an sdl based visualiser using imgui
type SdlImgui struct {
	// the mechanical requirements for the gui
	io      imgui.IO
	context *imgui.Context
	plt     *platform
	glsl    *glsl

	// references to the emulation
	lz *lazyvalues.Lazy
	tv television.Television

	// terminal interface to the debugger
	term *term

	// implementations of screen television protocols
	screen *screen
	audio  *sdlaudio.Audio

	// imgui window management
	wm *windowManager

	// the colors used by the imgui system. includes the TV colors in a
	// suitable format
	cols *imguiColors

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

	// is emulation running
	paused bool

	// mouse coords at last frame
	mx, my int32

	// the preferences we'll be saving to disk
	prefs *prefs.Disk
}

// NewSdlImgui is the preferred method of initialisation for type SdlImgui
//
// MUST ONLY be called from the #mainthread
func NewSdlImgui(tv television.Television) (*SdlImgui, error) {
	img := &SdlImgui{
		context:    imgui.CreateContext(nil),
		io:         imgui.CurrentIO(),
		tv:         tv,
		service:    make(chan func(), 1),
		serviceErr: make(chan error, 1),
		featureReq: make(chan featureRequest, 1),
		featureErr: make(chan error, 1),
	}

	var err error

	// define colors
	img.cols = newColors()

	img.plt, err = newPlatform(img)
	if err != nil {
		return nil, errors.New(errors.SDLImgui, err)
	}

	img.glsl, err = newGlsl(img.io, img)
	if err != nil {
		return nil, errors.New(errors.SDLImgui, err)
	}

	iniPath, err := paths.ResourcePath("", imguiIniFile)
	if err != nil {
		return nil, errors.New(errors.SDLImgui, err)
	}
	img.io.SetIniFilename(iniPath)

	// we don't have access to the Debugger, Disassembly or the VCS yet. those
	// fields in the lazy instance will be set when the requests come in
	img.lz = lazyvalues.NewValues()

	img.screen = newScreen(img)
	img.term = newTerm()

	img.wm, err = newWindowManager(img)
	if err != nil {
		return nil, errors.New(errors.SDLImgui, err)
	}

	// connect some screen properties to other parts of the system
	img.glsl.screenTextureID = img.screen.screenTexture
	tv.AddPixelRenderer(img.screen)

	// this audio mixer produces the sound. there is another AudioMixer
	// implementation in winAudio which visualises the sound
	img.audio, err = sdlaudio.NewAudio()
	if err != nil {
		return nil, errors.New(errors.SDLImgui, err)
	}
	tv.AddAudioMixer(img.audio)

	// setup preferences
	pth, err := paths.ResourcePath("", prefs.DefaultPrefsFile)
	if err != nil {
		return nil, errors.New(errors.SDLImgui, err)
	}
	img.prefs, err = prefs.NewDisk(pth)

	err = img.prefs.Add("sdlimgui.debugger.windowsize", prefs.NewGeneric(
		func(s string) error {
			var w, h int32
			_, err := fmt.Sscanf(s, "%d,%d", &w, &h)
			if err != nil {
				return err
			}
			img.plt.window.SetSize(w, h)
			return nil
		},
		func() string {
			w, h := img.plt.window.GetSize()
			return fmt.Sprintf("%d,%d", w, h)
		},
	))
	if err != nil {
		return nil, err
	}

	err = img.prefs.Add("sdlimgui.debugger.windowpos", prefs.NewGeneric(
		func(s string) error {
			var x, y int32
			_, err := fmt.Sscanf(s, "%d,%d", &x, &y)
			if err != nil {
				return err
			}
			// !TODO: SetPosition doesn't seem to set window position as you
			// might expect. On XWindow with Cinnamon WM, it seems to place the
			// window top to the window further down and slightly to the right
			// of where it should be. This means that the window "drifts" down
			// the screen on subsequent loads
			img.plt.window.SetPosition(x, y)
			return nil
		},
		func() string {
			x, y := img.plt.window.GetPosition()
			return fmt.Sprintf("%d,%d", x, y)
		},
	))
	if err != nil {
		return nil, errors.New(errors.SDLImgui, err)
	}

	// load preferences from disk
	err = img.prefs.Load()
	if err != nil {
		// ignore missing prefs file errors
		if !errors.Is(err, errors.PrefsNoFile) {
			return nil, errors.New(errors.SDLImgui, err)
		}
	}

	img.plt.window.Show()

	return img, nil
}

// Destroy implements GuiCreator interface
//
// MUST ONLY be called from the #mainthread
func (img *SdlImgui) Destroy(output io.Writer) {
	// we don't want to save preferences if we're in the middle of a panic
	if r := recover(); r == nil {
		_ = img.prefs.Save()
	}

	img.wm.destroy()
	img.audio.EndMixing()
	img.glsl.destroy()

	err := img.plt.destroy()
	if err != nil {
		output.Write([]byte(err.Error()))
	}

	img.context.Destroy()
}

// GetTerminal implements terminal.Broker interface
func (img *SdlImgui) GetTerminal() terminal.Terminal {
	return img.term
}

func (img *SdlImgui) pause(set bool) {
	img.paused = set
}

func (img *SdlImgui) draw() {
	if img.lz.Dbg == nil {
	} else {
		img.wm.draw()
	}
}
