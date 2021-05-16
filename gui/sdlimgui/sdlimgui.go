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
	"github.com/jetsetilly/gopher2600/debugger"
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
	"github.com/jetsetilly/gopher2600/userinput"
	"github.com/veandco/go-sdl2/sdl"

	"github.com/inkyblackness/imgui-go/v4"
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

	// lazy value system allows safe access to the debugger/emulation from the
	// GUI thread
	lz *lazyvalues.LazyValues

	// the gui renders differently depending on EmulationState. use setState()
	// to set the value
	state gui.EmulationState

	// isRewindSlider records whether the rewind slider control in the
	// winControl type is being clicked/moved.
	//
	// we can think of this as a special case of gui.State.Rewinding.
	isRewindSlider bool

	// vcs is set by ReqSetPlaymode or ReqSetDebugmode. in debug mode the VCS
	// is accessible via lz.Dbg.VCS but for maximum compatibility between
	// playmode and debugmode the VCS should be addressed through this pointer
	// where possible
	vcs *hardware.VCS

	// television sends the GUI information through the PixelRenderer and
	// AudioMixer.
	tv *television.Television

	// terminal interface to the debugger. this is distinct from the
	// winTerminal type.
	term *term

	// implementations of television protocols, PixelRenderer and AudioMixer
	screen *screen
	audio  *sdlaudio.Audio

	// the playscreen is drawn to the background of the platform window
	playScr *playScr

	// imgui window management
	wm *manager

	// the colors used by the imgui system. includes the TV colors converted to
	// a suitable format
	cols *imguiColors

	// polling encapsulates the programmatic communication to the service loop.
	// how the feature requests, pushed functions etc. are handled by the
	// service loop is important to the GUI's responsiveness.
	polling *polling

	// userinput channel is not created but assigned with ReqSetPlaymode and
	// ReqSetDebugmode requests
	userinput chan userinput.Event

	// mouse coords at last frame. used by service loop to keep track of mouse motion
	mouseX, mouseY int32

	// gui specific preferences. crt preferences are handled separately. all
	// other preferences are handled by the emulation
	prefs    *preferences
	crtPrefs *crt.Preferences

	// hasModal should be true for the duration of when a modal popup is on the screen
	hasModal bool

	// a request for the PlusROM first installation procedure has been received
	plusROMFirstInstallation *gui.PlusROMFirstInstallation
}

// NewSdlImgui is the preferred method of initialisation for type SdlImgui
//
// MUST ONLY be called from the gui thread.
func NewSdlImgui(tv *television.Television) (*SdlImgui, error) {
	img := &SdlImgui{
		context: imgui.CreateContext(nil),
		io:      imgui.CurrentIO(),
		tv:      tv,
	}

	var err error

	// define colors
	img.cols = newColors()

	img.plt, err = newPlatform(img)
	if err != nil {
		return nil, curated.Errorf("sdlimgui: %v", err)
	}

	img.glsl, err = newGlsl(img)
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

	img.playScr = newPlayScr(img)

	img.wm, err = newManager(img)
	if err != nil {
		return nil, curated.Errorf("sdlimgui: %v", err)
	}

	// initialise new polling type
	img.polling = newPolling(img)

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

	// open container window
	img.plt.window.Show()

	return img, nil
}

// Destroy implements GuiCreator interface
//
// MUST ONLY be called from the gui thread.
func (img *SdlImgui) Destroy(output io.Writer) {
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
	img.playScr.draw()
	img.wm.draw()
	img.drawPlusROMFirstInstallation()
}

// set emulation state and handle any changes.
func (img *SdlImgui) setEmulationState(state gui.EmulationState) {
	img.state = state

	switch img.state {
	case gui.StateInitialising:
		img.lz.SetActive(false)
	default:
		img.lz.SetActive(true)
	}

	switch img.state {
	case gui.StatePaused:
		img.screen.render()
	case gui.StateRunning:
		img.screen.render()
	case gui.StateEnding:
		img.prefs.saveWin()
	}
}

// is the gui in playmode or not. thread safe. called from emulation thread
// and gui thread.
func (img *SdlImgui) isPlaymode() bool {
	return img.lz.Dbg == nil
}

// set debugger/VCS and handle the changeover gracefully. this includes the saving
// and loading of preference groups. should only be called from gui thread.
func (img *SdlImgui) setDbgAndVCS(dbg *debugger.Debugger, vcs *hardware.VCS) error {
	img.lz.Dbg = dbg

	// playmode requesed
	if dbg == nil {
		img.vcs = vcs

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

		img.screen.clearTextureRenderers()
		img.screen.addTextureRenderer(img.playScr)

		return nil
	}

	// debugging mode requested
	img.vcs = dbg.VCS

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

	img.screen.clearTextureRenderers()
	img.screen.addTextureRenderer(img.wm.dbgScr)

	return nil
}

// has mouse been grabbed. only called from gui thread.
func (img *SdlImgui) isCaptured() bool {
	if img.isPlaymode() {
		return img.playScr.isCaptured
	}
	return img.wm.dbgScr.isCaptured
}

// grab mouse. only called from gui thread.
func (img *SdlImgui) setCapture(set bool) {
	if img.isPlaymode() {
		img.playScr.isCaptured = set
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
