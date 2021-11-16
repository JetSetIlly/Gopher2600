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
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/gui/crt"
	"github.com/jetsetilly/gopher2600/gui/sdlaudio"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/lazyvalues"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/reflection"
	"github.com/jetsetilly/gopher2600/resources"
	"github.com/jetsetilly/gopher2600/userinput"
	"github.com/veandco/go-sdl2/sdl"

	"github.com/inkyblackness/imgui-go/v4"
)

// imguiIniFile is where imgui will store the coordinates of the imgui windows
const imguiIniFile = "debugger_imgui.ini"

// SdlImgui is an sdl based visualiser using imgui.
type SdlImgui struct {
	// the mechanical requirements for the gui
	io      imgui.IO
	context *imgui.Context
	plt     *platform
	glsl    *glsl

	// parent emulation. should be set via setEmulation() only
	emulation emulation.Emulation

	// the current mode of the underlying
	mode emulation.Mode

	// taken from the emulation field and assigned in the setEmulation() function
	tv        *television.Television
	vcs       *hardware.VCS
	dbg       *debugger.Debugger
	userinput chan userinput.Event

	// lazy value system allows safe access to the debugger/emulation from the
	// GUI thread
	lz *lazyvalues.LazyValues

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
func NewSdlImgui(e emulation.Emulation) (*SdlImgui, error) {
	img := &SdlImgui{
		context: imgui.CreateContext(nil),
		io:      imgui.CurrentIO(),
	}

	img.emulation = e
	img.tv = e.TV().(*television.Television)
	img.vcs = e.VCS().(*hardware.VCS)
	switch dbg := e.Debugger().(type) {
	case *debugger.Debugger:
		img.dbg = dbg
	}
	img.userinput = e.UserInput()

	// path to dear imgui ini file
	iniPath, err := resources.JoinPath(imguiIniFile)
	if err != nil {
		return nil, curated.Errorf("sdlimgui: %v", err)
	}
	img.io.SetIniFilename(iniPath)

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

	img.lz = lazyvalues.NewLazyValues(img.emulation)
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
	img.tv.AddPixelRenderer(img.screen)

	// texture renderers are added depending on playmode

	// this audio mixer produces the sound. there is another AudioMixer
	// implementation in winAudio which visualises the sound
	img.audio, err = sdlaudio.NewAudio()
	if err != nil {
		return nil, curated.Errorf("sdlimgui: %v", err)
	}
	img.tv.AddRealtimeAudioMixer(img.audio)

	// load sdlimgui preferences
	img.prefs, err = newPreferences(img)
	if err != nil {
		return nil, curated.Errorf("sdlimgui: %v", err)
	}

	// initialise crt preferences
	img.crtPrefs, err = crt.NewPreferences()
	if err != nil {
		return nil, curated.Errorf("sdlimgui: %v", err)
	}

	// container window is open on setEmulationMode()

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

// GetReflectionRenderer implements reflection.Broker interface.
func (img *SdlImgui) GetReflectionRenderer() reflection.Renderer {
	return img.screen
}

// quit application sends a request to the emulation.
func (img *SdlImgui) quit() {
	if !img.hasModal {
		select {
		case img.userinput <- userinput.EventQuit{}:
		default:
			logger.Log("sdlimgui", "dropped quit event")
		}
	}
}

// end program. this differs from quit in that this function is called when we
// receive a ReqEnd, which *may* have been sent in reponse to a EventQuit.
func (img *SdlImgui) end() {
	img.prefs.saveWindowPreferences()
}

// draw gui. called from service loop.
func (img *SdlImgui) draw() {
	if img.emulation.State() == emulation.EmulatorStart {
		return
	}

	if img.mode == emulation.ModePlay {
		img.playScr.draw()
	}

	img.wm.draw()
	img.drawPlusROMFirstInstallation()
}

// is the gui in playmode or not. thread safe. called from emulation thread
// and gui thread.
func (img *SdlImgui) isPlaymode() bool {
	return img.mode == emulation.ModePlay
}

// set emulation and handle the changeover gracefully. this includes the saving
// and loading of preference groups.
//
// should only be called from gui thread.
func (img *SdlImgui) setEmulationMode(mode emulation.Mode) error {
	img.mode = mode
	img.prefs.loadWindowPreferences()

	switch mode {
	case emulation.ModeDebugger:
		img.screen.clearTextureRenderers()
		img.screen.addTextureRenderer(img.wm.dbgScr)

		err := img.prefs.load()
		if err != nil {
			return err
		}

		img.plt.window.Show()

	case emulation.ModePlay:
		img.screen.clearTextureRenderers()
		img.screen.addTextureRenderer(img.playScr)

		err := img.prefs.load()
		if err != nil {
			return err
		}

		img.plt.window.Show()
	}

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
