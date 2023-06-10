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
	"sync/atomic"
	"time"

	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/gui/crt"
	"github.com/jetsetilly/gopher2600/gui/sdlaudio"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/lazyvalues"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/notifications"
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/reflection"
	"github.com/jetsetilly/gopher2600/resources"
	"github.com/jetsetilly/gopher2600/userinput"
	"github.com/veandco/go-sdl2/sdl"

	"github.com/inkyblackness/imgui-go/v4"
)

// imguiIniFile is where imgui will store the coordinates of the imgui windows
const imguiIniFile = "debugger_imgui.ini"

// the number of frames to count before resetting fonts
const resetFontFrames = 2

// the amount to fade a widget by when disabled
const disabledAlpha = 0.3

// SdlImgui is an sdl based visualiser using imgui.
type SdlImgui struct {
	// the mechanical requirements for the gui
	plt  *platform
	glsl *glsl

	// resetFonts value will be reduced to 0 each GUI frame. at value 1 the
	// fonts will be reset.
	//
	// set to 1 to reset immediately and higher values to delay the reset by
	// that number of frames. higher values are useful in the prefs window to
	// allow sufficient time to close the font size selector combo box
	resetFonts int

	// the current mode of the underlying
	mode atomic.Value // govern.Mode

	// references to parent emulation
	dbg       *debugger.Debugger
	tv        *television.Television
	vcs       *hardware.VCS
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

	// the time the window was focused. we use this to prevent the TAB key from
	// opening the ROM requester if the window was focused with the alt-TAB key
	// combo used by Windows and many Linux desktops
	windowFocusedTime time.Time

	// gui specific preferences. crt preferences are handled separately. all
	// other preferences are handled by the emulation
	prefs    *preferences
	crtPrefs *crt.Preferences

	// hasModal should be true for the duration of when a modal popup is on the screen
	hasModal bool

	// a request for the PlusROM first installation procedure has been received
	plusROMFirstInstallation bool

	// functions that should only be run after gui rendering
	postRenderFunctions chan func()
}

// NewSdlImgui is the preferred method of initialisation for type SdlImgui
//
// MUST ONLY be called from the gui thread.
func NewSdlImgui(dbg *debugger.Debugger) (*SdlImgui, error) {
	img := &SdlImgui{
		dbg:                 dbg,
		tv:                  dbg.TV(),
		vcs:                 dbg.VCS(),
		userinput:           dbg.UserInput(),
		postRenderFunctions: make(chan func(), 100),
	}

	// create imgui context
	imgui.CreateContext(nil)

	// mode is in the none state at the beginning
	img.mode.Store(govern.ModeNone)

	// path to dear imgui ini file
	iniPath, err := resources.JoinPath(imguiIniFile)
	if err != nil {
		return nil, fmt.Errorf("sdlimgui: %w", err)
	}
	imgui.CurrentIO().SetIniFilename(iniPath)

	// define colors
	img.cols = newColors()

	img.plt, err = newPlatform(img)
	if err != nil {
		return nil, fmt.Errorf("sdlimgui: %w", err)
	}

	// initialise preferences after platform initialisation because there are
	// preference hooks that reference platform fields
	img.prefs, err = newPreferences(img)
	if err != nil {
		return nil, fmt.Errorf("sdlimgui: %w", err)
	}

	img.glsl, err = newGlsl(img)
	if err != nil {
		return nil, fmt.Errorf("sdlimgui: %w", err)
	}

	img.lz = lazyvalues.NewLazyValues(img.dbg)
	img.screen = newScreen(img)
	img.term = newTerm()

	img.playScr = newPlayScr(img)

	img.wm, err = newManager(img)
	if err != nil {
		return nil, fmt.Errorf("sdlimgui: %w", err)
	}

	err = img.wm.loadManagerState()
	if err != nil {
		return nil, fmt.Errorf("sdlimgui: %w", err)
	}

	err = img.wm.loadManagerHotkeys()
	if err != nil {
		return nil, fmt.Errorf("sdlimgui: %w", err)
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
		return nil, fmt.Errorf("sdlimgui: %w", err)
	}
	img.tv.AddRealtimeAudioMixer(img.audio)

	// initialise crt preferences
	img.crtPrefs, err = crt.NewPreferences()
	if err != nil {
		return nil, fmt.Errorf("sdlimgui: %w", err)
	}

	// set scaling when IntegerScaling prefs value is changed
	img.crtPrefs.IntegerScaling.SetHookPost(func(v prefs.Value) error {
		select {
		case img.postRenderFunctions <- func() {
			img.screen.crit.section.Lock()
			img.playScr.setScaling()
			img.screen.crit.section.Unlock()
		}:
		default:
		}
		return nil
	})

	// set event filter for SDL see comment for serviceWindowEvent()
	sdl.AddEventWatchFunc(img.serviceWindowEvent, nil)

	// reset fonts as soon as possible
	img.resetFonts = 1

	// container window is open on setEmulationMode()

	// load preferences
	err = img.prefs.load()
	if err != nil {
		return nil, err
	}

	return img, nil
}

// Destroy implements GuiCreator interface
//
// MUST ONLY be called from the gui thread.
func (img *SdlImgui) Destroy() {
	err := img.prefs.saveOnExitDsk.Save()
	if err != nil {
		logger.Log("sdlimgui", err.Error())
	}

	err = img.audio.EndMixing()
	if err != nil {
		logger.Log("sdlimgui", err.Error())
	}

	err = img.wm.saveManagerState()
	if err != nil {
		logger.Log("sdlimgui", err.Error())
	}

	err = img.wm.saveManagerHotkeys()
	if err != nil {
		logger.Log("sdlimgui", err.Error())
	}

	err = img.plt.destroy()
	if err != nil {
		logger.Log("sdlimgui", err.Error())
	}

	// destroying glsl after platform or we'll get a panic when window is
	// opened in full screen
	img.glsl.destroy()

	ctx, err := imgui.CurrentContext()
	if err != nil {
		logger.Log("sdlimgui", err.Error())
	}
	ctx.Destroy()
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
	mode := img.mode.Load().(govern.Mode)

	if mode == govern.ModeNone {
		return
	}

	if img.dbg.State() == govern.EmulatorStart {
		return
	}

	imgui.PushFont(img.glsl.fonts.defaultFont)
	defer imgui.PopFont()

	if mode == govern.ModePlay {
		img.playScr.draw()
	}

	img.wm.draw()
	img.drawPlusROMFirstInstallation()
}

// is the gui in playmode or not. thread safe. called from emulation thread
// and gui thread.
func (img *SdlImgui) isPlaymode() bool {
	return img.mode.Load().(govern.Mode) == govern.ModePlay
}

// set emulation and handle the changeover gracefully. this includes the saving
// and loading of preference groups.
//
// should only be called from gui thread.
func (img *SdlImgui) setEmulationMode(mode govern.Mode) error {
	// release captured mouse before switching emulation modes. if we don't do
	// this then the capture state will remain if we flip back to the emulation
	// mode later. at this point the captured mouse can cause confusion
	img.setCapture(false)

	img.mode.Store(mode)
	img.prefs.loadWindowPreferences()

	switch mode {
	case govern.ModeDebugger:
		img.screen.clearTextureRenderers()
		img.screen.addTextureRenderer(img.wm.dbgScr)
		img.plt.window.Show()
		img.plt.window.Raise()

	case govern.ModePlay:
		img.screen.clearTextureRenderers()
		img.screen.addTextureRenderer(img.playScr)
		img.plt.window.Show()
		img.plt.window.Raise()
	}

	img.setAudioMute()

	// small delay before calling smartHideCursor(). this is because SDL will
	// detect a mouse motion on startup or when the window changes between
	// emulation modes. rather than dealing with thresholds in the event
	// handler, it's easy and cleaner to set a short delay
	time.AfterFunc(250*time.Millisecond, func() {
		img.smartHideCursor(true)
	})

	return nil
}

func (img *SdlImgui) toggleAudioMute() {
	if img.isPlaymode() {
		m := img.prefs.audioMutePlaymode.Get().(bool)
		img.prefs.audioMutePlaymode.Set(!m)
	} else {
		m := img.prefs.audioMuteDebugger.Get().(bool)
		img.prefs.audioMuteDebugger.Set(!m)
	}

	// the act of setting the prefs value means that setAudioMute() is called
	// indirectly, so there's no need to call it here
}

func (img *SdlImgui) setAudioMute() {
	// if there is no prefs instance then return without error. this can happen
	// when the prefs are being loaded from disk for the first time and the
	// prefs instance hasn't been returned
	if img.prefs == nil {
		return
	}

	var mute bool

	if img.isPlaymode() {
		mute = img.prefs.audioMutePlaymode.Get().(bool)
		if mute {
			img.playScr.emulationNotice.set(notifications.NotifyMute)
		} else {
			img.playScr.emulationNotice.set(notifications.NotifyUnmute)
		}
		img.vcs.RIOT.Ports.MutePeripherals(mute)
	} else {
		mute = img.prefs.audioMuteDebugger.Get().(bool)
		img.vcs.RIOT.Ports.MutePeripherals(mute)
	}

	img.audio.Mute(mute)
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
		if set {
			img.wm.dbgScr.debuggerSetOpen(true)
			img.wm.dbgScr.debuggerGeometry().raise = true
		}
		img.wm.dbgScr.isCaptured = set
	}

	err := sdl.CaptureMouse(set)
	if err != nil {
		logger.Log("sdlimgui", err.Error())
	}

	img.plt.window.SetGrab(set)
	img.hideCursor(set)
}

// "captured running" is when the emulation is running inside the debugger and
// the input has been captured
//
// always returns false if the emulation is in playmode
func (img *SdlImgui) isCapturedRunning() bool {
	if img.isPlaymode() {
		return false
	}
	return img.wm.dbgScr.isCaptured && img.dbg.State() == govern.Running
}

// set "captured running". does nothing if the emulation is in playmode
func (img *SdlImgui) setCapturedRunning(set bool) {
	if img.isPlaymode() {
		return
	}

	if set {
		img.setCapture(true)
		img.term.pushCommand("RUN")
	} else {
		img.setCapture(false)
		img.term.pushCommand("HALT")
		if img.wm.refocusWindow != nil {
			geom := img.wm.refocusWindow.debuggerGeometry()
			geom.raise = true
		}
	}
}

// smartly hide cursor based on playmode and capture state. only called from
// gui thread
func (img *SdlImgui) smartHideCursor(set bool) {
	if img.isPlaymode() && !img.isCaptured() {
		img.hideCursor(set)
	}
}

// hide mouse cursor. only called from gui thread
func (img *SdlImgui) hideCursor(hide bool) {
	if hide {
		_, err := sdl.ShowCursor(sdl.DISABLE)
		if err != nil {
			logger.Log("sdlimgui", err.Error())
		}
	} else {
		_, err := sdl.ShowCursor(sdl.ENABLE)
		if err != nil {
			logger.Log("sdlimgui", err.Error())
		}
	}
}
