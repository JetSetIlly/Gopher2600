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
	"io"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/gui/sdlaudio"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/lazyvalues"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/paths"
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/reflection"

	"github.com/inkyblackness/imgui-go/v2"
)

// imguiIniFile is where imgui will store the coordinates of the imgui windows
// !!TODO: duplicate imgui.SetIniFilename so that is uses prefs package. we
// should be able to do this a smart implementation of io.Reader and io.Writer.
const imguiIniFile = "debugger_imgui.ini"

// SdlImgui is an sdl based visualiser using imgui.
type SdlImgui struct {
	// the mechanical requirements for the gui
	io      imgui.IO
	context *imgui.Context
	plt     *platform
	glsl    *glsl

	// references to the emulation
	lz  *lazyvalues.LazyValues
	tv  *television.Television
	vcs *hardware.VCS

	// terminal interface to the debugger
	term *term

	// implementations of screen television protocols
	screen *screen
	audio  *sdlaudio.Audio

	// imgui window management
	wm *windowManager

	// the colors used by the imgui system. includes the TV colors converted to
	// a suitable format
	cols *imguiColors

	// functions that need to be performed in the main thread should be queued
	// for service
	service    chan func()
	serviceErr chan error

	// ReqFeature() hands off requests to the featureReq channel for servicing.
	// think of this as a special instance of the service chan
	featureReq chan featureRequest
	featureErr chan error

	// events channel is not created but assigned with the feature request
	// gui.ReqSetEventChan
	events chan gui.Event

	// is emulation running
	paused bool

	// mouse coords at last frame
	mx, my int32

	// the preferences we'll be saving to disk
	prefs *prefs.Disk

	// hasModal should be true for the duration of when a modal popup is on the screen
	hasModal bool

	// a request for the PlusROM first installation procedure has been received
	plusROMFirstInstallation *gui.PlusROMFirstInstallation
}

// NewSdlImgui is the preferred method of initialisation for type SdlImgui
//
// MUST ONLY be called from the #mainthread.
func NewSdlImgui(tv *television.Television, playmode bool) (*SdlImgui, error) {
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
		return nil, curated.Errorf("sdlimgui: %v", err)
	}

	img.glsl, err = newGlsl(img.io, img)
	if err != nil {
		return nil, curated.Errorf("sdlimgui: %v", err)
	}

	iniPath, err := paths.ResourcePath("", imguiIniFile)
	if err != nil {
		return nil, curated.Errorf("sdlimgui: %v", err)
	}
	img.io.SetIniFilename(iniPath)

	img.lz = lazyvalues.NewLazyValues()
	img.screen = newScreen(img)
	img.term = newTerm()

	img.wm, err = newWindowManager(img)
	if err != nil {
		return nil, curated.Errorf("sdlimgui: %v", err)
	}

	// connect pixel renderer/referesher to television and texture renderer to
	// pixel renderer
	tv.AddPixelRenderer(img.screen)
	img.screen.addTextureRenderer(img.wm.dbgScr)
	img.screen.addTextureRenderer(img.wm.playScr)

	// this audio mixer produces the sound. there is another AudioMixer
	// implementation in winAudio which visualises the sound
	img.audio, err = sdlaudio.NewAudio()
	if err != nil {
		return nil, curated.Errorf("sdlimgui: %v", err)
	}
	tv.AddAudioMixer(img.audio)

	// initialise debugger preferences. in the event of playmode being set this
	// will immediately be replaced but frankly doing it this way is cleaner
	err = img.initPrefs(prefsGrpDebugger)
	if err != nil {
		return nil, curated.Errorf("sdlimgui: %v", err)
	}

	// set playmode according to the playmode argument
	err = img.setPlaymode(playmode)
	if err != nil {
		return nil, curated.Errorf("sdlimgui: %v", err)
	}

	// open container window
	img.plt.window.Show()

	return img, nil
}

// Destroy implements GuiCreator interface
//
// MUST ONLY be called from the #mainthread.
func (img *SdlImgui) Destroy(output io.Writer) {
	img.wm.destroy()
	err := img.audio.EndMixing()
	if err != nil {
		output.Write([]byte(err.Error()))
	}
	img.glsl.destroy()

	err = img.plt.destroy()
	if err != nil {
		output.Write([]byte(err.Error()))
	}

	img.context.Destroy()
}

func (img *SdlImgui) draw() {
	img.wm.draw()
	img.drawPlusROMFirstInstallation()
}

// GetTerminal implements terminal.Broker interface.
func (img *SdlImgui) GetTerminal() terminal.Terminal {
	return img.term
}

// GetReflectionRendere implements reflection.Broker interface.
func (img *SdlImgui) GetReflectionRenderer() reflection.Renderer {
	return img.screen
}

// the following functions are used to differentiate play-mode from debug-mode.
// any operation that is dependent on playmode state should be abstracted to a
// function and placed below.
//
// for simplicity, play-mode is defined as being on when playScr is open

func (img *SdlImgui) isPlaymode() bool {
	return img.wm != nil && img.wm.playScr.isOpen()
}

// set playmode and handle the changeover gracefully. this includes the saving
// and loading of preference groups.
func (img *SdlImgui) setPlaymode(set bool) error {
	if set {
		if !img.isPlaymode() {
			if img.prefs != nil {
				err := img.prefs.Save()
				if err != nil {
					return err
				}
			}
			err := img.initPrefs(prefsGrpPlaymode)
			if err != nil {
				return err
			}

			img.wm.playScr.setOpen(true)
		}
	} else {
		if img.isPlaymode() {
			if img.prefs != nil {
				if err := img.prefs.Save(); err != nil {
					return err
				}
			}
			err := img.initPrefs(prefsGrpDebugger)
			if err != nil {
				return err
			}
			img.wm.playScr.setOpen(false)
		}
	}

	return nil
}

func (img *SdlImgui) isCaptured() bool {
	if img.isPlaymode() {
		return img.wm.playScr.isCaptured
	}
	return img.wm.dbgScr.isCaptured
}

func (img *SdlImgui) setCapture(set bool) {
	if img.isPlaymode() {
		img.wm.playScr.isCaptured = set
		return
	}
	img.wm.dbgScr.isCaptured = set
}

func (img *SdlImgui) isHovered() bool {
	if img.isPlaymode() {
		return true
	}
	return img.wm.dbgScr.isHovered && !img.wm.dbgScr.isPopup
}

// scaling of the tv screen also depends on whether playmode is active

type scalingScreen interface {
	getScaling(horiz bool) float32
	setScaling(scaling float32)
}

func (img *SdlImgui) setScale(scaling float32, adjust bool) {
	var scr scalingScreen

	if img.isPlaymode() {
		scr = img.wm.playScr
	} else {
		scr = img.wm.dbgScr
	}

	if adjust {
		scale := scr.getScaling(false)
		if scale > 0.5 && scale < 4.0 {
			scr.setScaling(scale + scaling)
		}
	} else {
		scr.setScaling(scaling)
	}
}
