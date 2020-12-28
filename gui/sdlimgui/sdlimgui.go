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
	"sync/atomic"
	"time"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/gui/crt"
	"github.com/jetsetilly/gopher2600/gui/sdlaudio"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/lazyvalues"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/paths"
	"github.com/jetsetilly/gopher2600/reflection"
	"github.com/veandco/go-sdl2/sdl"

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

	// is gui in playmode. use setPlaymode() and isPlaymode() to access this
	playmode atomic.Value

	// the gui renders differently depending on EmulationState. use setState()
	// to set the value
	state gui.EmulationState

	// terminal interface to the debugger
	term *term

	// implementations of television protocols, PixelRenderer and AudioMixer
	screen *screen
	audio  *sdlaudio.Audio

	// imgui window management
	wm *manager

	// the colors used by the imgui system. includes the TV colors converted to
	// a suitable format
	cols *imguiColors

	// functions that need to be performed in the main thread are queued for
	// serving by the service() function
	service           chan func()
	serviceErr        chan error
	servicePulsePlay  *time.Ticker
	servicePulseDebug *time.Ticker
	servicePulseIdle  *time.Ticker

	// some gui events will not be serviced immediately because of the service
	// sleep. serviceWake causes the service loop to wake up immediately.
	//
	// when pushing to this channel from the same goroutine as the service loop
	// (which is most likely) then the push should happen in a select/default
	// block to prevent channel deadlock. eg:
	//
	//	select {
	//	case serviceWake <- true:
	//	default:
	//	}
	serviceWake chan bool

	// ReqFeature() and GetFeature() hands off requests to the featureReq
	// channel for servicing. think of these as pecial instances of the
	// service chan
	featureSet     chan featureRequest
	featureSetErr  chan error
	featureGet     chan featureRequest
	featureGetData chan gui.FeatureReqData
	featureGetErr  chan error

	// events channel is not created but assigned with the feature request
	// gui.ReqSetEventChan. it is a way for the gui to send information about
	// events back to the emulation.
	events chan gui.Event

	// mouse coords at last frame. used by service loop to keep track of mouse motion
	mouseX, mouseY int32

	// gui specific preferences. crt preferences are handled separately. all
	// other preferences are handled by the emulation
	prefs    *Preferences
	crtPrefs *crt.Preferences

	// hasModal should be true for the duration of when a modal popup is on the screen
	hasModal bool

	// a request for the PlusROM first installation procedure has been received
	plusROMFirstInstallation *gui.PlusROMFirstInstallation
}

// NewSdlImgui is the preferred method of initialisation for type SdlImgui
//
// MUST ONLY be called from the gui thread.
func NewSdlImgui(tv *television.Television, playmode bool) (*SdlImgui, error) {
	img := &SdlImgui{
		context:        imgui.CreateContext(nil),
		io:             imgui.CurrentIO(),
		tv:             tv,
		service:        make(chan func(), 1),
		serviceErr:     make(chan error, 1),
		featureSet:     make(chan featureRequest, 1),
		featureSetErr:  make(chan error, 1),
		featureGet:     make(chan featureRequest, 1),
		featureGetData: make(chan gui.FeatureReqData, 1),
		featureGetErr:  make(chan error, 1),
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

	img.wm, err = newManager(img)
	if err != nil {
		return nil, curated.Errorf("sdlimgui: %v", err)
	}

	// connect pixel renderer/referesher to television and texture renderer to
	// pixel renderer
	tv.AddPixelRenderer(img.screen)

	// texture renderers are added depending on playmode

	// this audio mixer produces the sound. there is another AudioMixer
	// implementation in winAudio which visualises the sound
	img.audio, err = sdlaudio.NewAudio()
	if err != nil {
		return nil, curated.Errorf("sdlimgui: %v", err)
	}
	tv.AddAudioMixer(img.audio)

	// initialise crt preferences
	img.crtPrefs, err = crt.NewPreferences()
	if err != nil {
		return nil, curated.Errorf("sdlimgui: %v", err)
	}

	// initialise service pulses. which one we're using depends on the state of
	// the gui (whether it's in playmode etc.)
	img.servicePulseDebug = time.NewTicker(time.Millisecond * debugSleepPeriod)
	img.servicePulsePlay = time.NewTicker(time.Millisecond * playSleepPeriod)
	img.servicePulseIdle = time.NewTicker(time.Millisecond * idleSleepPeriod)

	// channel to force service loop to wake from a delay
	img.serviceWake = make(chan bool, 1)

	// playmode is an atomic value. make sure a value has been assigned to it
	// before accessing it.
	img.playmode.Store(false)

	// set playmode according to the playmode argument. this will set and load
	// the correct preferences
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
// MUST ONLY be called from the gui thread.
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

// GetTerminal implements terminal.Broker interface.
func (img *SdlImgui) GetTerminal() terminal.Terminal {
	return img.term
}

// GetReflectionRendere implements reflection.Broker interface.
func (img *SdlImgui) GetReflectionRenderer() reflection.Renderer {
	return img.screen
}

// draw gui. called from service loop.
func (img *SdlImgui) draw() {
	img.wm.draw()
	img.drawPlusROMFirstInstallation()
}

// set emulation state and handle any changes.
func (img *SdlImgui) setEmulationState(state gui.EmulationState) {
	img.state = state
	img.screen.render()
}

// is the gui in playmode or not. thread safe. called from emulation thread
// and gui thread.
func (img *SdlImgui) isPlaymode() bool {
	return img.playmode.Load().(bool)
}

// set playmode and handle the changeover gracefully. this includes the saving
// and loading of preference groups. should only be called from gui thread.
func (img *SdlImgui) setPlaymode(set bool) error {
	img.playmode.Store(set)

	if set {
		// save current preferences
		if img.prefs != nil {
			err := img.prefs.save()
			if err != nil {
				return err
			}
		}

		// load playmode preferences
		var err error
		img.prefs, err = newPlaymodePreferences(img)
		if err != nil {
			return curated.Errorf("sdlimgui: %v", err)
		}

		img.wm.playScr.setOpen(true)
		img.screen.clearTextureRenderers()
		img.screen.addTextureRenderer(img.wm.playScr)

		return nil
	}

	// debugging mode requested

	// save current preferences
	if img.prefs != nil {
		if err := img.prefs.save(); err != nil {
			return err
		}
	}

	// load debugging preferences
	var err error
	img.prefs, err = newDebugPreferences(img)
	if err != nil {
		return curated.Errorf("sdlimgui: %v", err)
	}

	img.wm.playScr.setOpen(false)
	img.screen.clearTextureRenderers()
	img.screen.addTextureRenderer(img.wm.dbgScr)

	return nil
}

// has mouse been grabbed. only called from gui thread.
func (img *SdlImgui) isCaptured() bool {
	if img.isPlaymode() {
		return img.wm.playScr.isCaptured
	}
	return img.wm.dbgScr.isCaptured
}

// grab mouse. only called from gui thread.
func (img *SdlImgui) setCapture(set bool) {
	if img.isPlaymode() {
		img.wm.playScr.isCaptured = set
	} else {
		img.wm.dbgScr.isCaptured = set
	}

	err := sdl.CaptureMouse(set)
	if err != nil {
		logger.Log("sdlimgui", err.Error())
	}

	img.plt.window.SetGrab(set)

	if set {
		_, err = sdl.ShowCursor(sdl.DISABLE)
		if err != nil {
			logger.Log("sdlimgui", err.Error())
		}
	} else {
		_, err = sdl.ShowCursor(sdl.ENABLE)
		if err != nil {
			logger.Log("sdlimgui", err.Error())
		}
	}
}
