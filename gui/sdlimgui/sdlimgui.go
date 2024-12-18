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
	"github.com/jetsetilly/gopher2600/gui/display"
	"github.com/jetsetilly/gopher2600/gui/sdlaudio"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui/caching"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/logger"
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

// the phantom input system allows windows to obey key presses without any
// specific widget being active
type phantomInput int

// list of valid phantomInput values
const (
	phantomInputNone phantomInput = iota
	phantomInputRune
	phantomInputBackSpace
)

// SdlImgui is an sdl based visualiser using imgui.
type SdlImgui struct {
	// the mechanical requirements for the gui
	plt   *platform
	rnd   renderer
	fonts fontAtlas

	// resetFonts value will be reduced to 0 each GUI frame. at value 1 the
	// fonts will be reset.
	//
	// set to 1 to reset immediately and higher values to delay the reset by
	// that number of frames. higher values are useful in the prefs window to
	// allow sufficient time to close the font size selector combo box
	resetFonts int

	// the current mode of the emulation
	mode atomic.Value // govern.Mode

	// references to parent emulation
	dbg *debugger.Debugger

	// cached values from the emulation
	cache caching.Cache

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

	// if true then a tooltip has been requested. reset after menu is drawn
	tooltipIndicator bool

	// tooltips will show regardless of show tooltip preferences
	tooltipForce bool

	// the colors used by the imgui system. includes the TV colors converted to
	// a suitable format
	cols *imguiColors

	// polling encapsulates the programmatic communication to the service loop.
	// how the feature requests, pushed functions etc. are handled by the
	// service loop is important to the GUI's responsiveness.
	polling *polling

	// the phantom input system allows windows to obey key presses without any
	// specific widget being active
	phantomInput     phantomInput
	phantomInputRune rune

	// the time the window was focused. we use this to prevent the TAB key from
	// opening the ROM requester if the window was focused with the alt-TAB key
	// combo used by Windows and many Linux desktops
	windowFocusedTime time.Time

	// gui specific preferences. crt preferences are handled separately. all
	// other preferences are handled by the emulation
	prefs        *preferences
	displayPrefs *display.Preferences

	// modal window
	modal modal

	// functions that should only be run after gui rendering
	postRenderFunctions chan func()
}

// NewSdlImgui is the preferred method of initialisation for type SdlImgui
//
// MUST ONLY be called from the gui thread.
func NewSdlImgui(dbg *debugger.Debugger) (*SdlImgui, error) {
	img := &SdlImgui{
		dbg:                 dbg,
		postRenderFunctions: make(chan func(), 100),
		cache:               caching.NewCache(),
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

	// new renderer
	img.rnd = newRenderer(img)

	// create platform
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

	// initialise display preferences
	img.displayPrefs, err = display.NewPreferences()
	if err != nil {
		return nil, fmt.Errorf("sdlimgui: %w", err)
	}

	// start renderer after platform and preferences
	err = img.rnd.start()
	if err != nil {
		return nil, fmt.Errorf("sdlimgui: %w", err)
	}

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
	img.dbg.VCS().TV.AddPixelRenderer(img.screen)

	// texture renderers are added depending on playmode

	// this audio mixer produces the sound. there is another AudioMixer
	// implementation in winAudio which visualises the sound
	img.audio, err = sdlaudio.NewAudio()
	if err != nil {
		return nil, fmt.Errorf("sdlimgui: %w", err)
	}
	img.dbg.VCS().TV.AddRealtimeAudioMixer(img.audio)

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
		logger.Log(logger.Allow, "sdlimgui", err)
	}

	err = img.audio.EndMixing()
	if err != nil {
		logger.Log(logger.Allow, "sdlimgui", err)
	}

	err = img.wm.saveManagerState()
	if err != nil {
		logger.Log(logger.Allow, "sdlimgui", err)
	}

	err = img.wm.saveManagerHotkeys()
	if err != nil {
		logger.Log(logger.Allow, "sdlimgui", err)
	}

	img.wm.destroy()

	err = img.plt.destroy()
	if err != nil {
		logger.Log(logger.Allow, "sdlimgui", err)
	}

	img.rnd.destroy()

	ctx, err := imgui.CurrentContext()
	if err != nil {
		logger.Log(logger.Allow, "sdlimgui", err)
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
	select {
	case img.dbg.UserInput() <- userinput.EventQuit{}:
	default:
		logger.Log(logger.Allow, "sdlimgui", "dropped quit event")
	}
}

// end program. this differs from quit in that this function is called when we
// receive a ReqEnd, which *may* have been sent in reponse to a userinput.EventQuit
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

	imgui.PushFont(img.fonts.gui)
	defer imgui.PopFont()

	if mode == govern.ModePlay {
		img.playScr.draw()
	}

	img.wm.draw()
	img.modalDraw()
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

	img.applyAudioMutePreference()

	// small delay before calling smartHideCursor(). this is because SDL will
	// detect a mouse motion on startup or when the window changes between
	// emulation modes. rather than dealing with thresholds in the event
	// handler, it's easy and cleaner to set a short delay
	time.AfterFunc(250*time.Millisecond, func() {
		img.smartCursorVisibility(true)
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

// apply the preference for audio mute according to the current playmode
func (img *SdlImgui) applyAudioMutePreference() {
	// if there is no prefs instance then return without error. this can happen
	// when the prefs are being loaded from disk for the first time and the
	// prefs instance hasn't been returned
	if img.prefs == nil {
		return
	}

	// mute any sound producing peripherals attached to the VCS. the call to
	// MutePeripherals() must be done in the emulation goroutine
	var mute bool
	if img.isPlaymode() {
		mute = img.prefs.audioMutePlaymode.Get().(bool)
	} else {
		mute = img.prefs.audioMuteDebugger.Get().(bool)
	}
	img.dbg.PushFunction(func() {
		img.dbg.VCS().RIOT.Ports.MutePeripherals(mute)
	})

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
			img.wm.dbgScr.debuggerGeometry().raiseOnNextDraw = true
		}
		img.wm.dbgScr.isCaptured = set
	}

	img.plt.setCapture(set)
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
			geom.raiseOnNextDraw = true
		}
	}
}

// set visiblity of cursor
func (img *SdlImgui) cursorVisibility(hidden bool) {
	if hidden {
		_, err := sdl.ShowCursor(sdl.DISABLE)
		if err != nil {
			logger.Log(logger.Allow, "sdlimgui", err)
		}
	} else {
		_, err := sdl.ShowCursor(sdl.ENABLE)
		if err != nil {
			logger.Log(logger.Allow, "sdlimgui", err)
		}
	}
}

// set the visibility of the cursor based on playmode and capture state
func (img *SdlImgui) smartCursorVisibility(hidden bool) {
	if img.isPlaymode() && !img.isCaptured() {
		img.cursorVisibility(hidden)
	} else {
		img.cursorVisibility(false)
	}
}

func (img *SdlImgui) setReasonableWindowConstraints() {
	winw, winh := img.plt.windowSize()
	winw *= 0.95
	winh *= 0.95
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{X: 300, Y: 300}, imgui.Vec2{X: winw, Y: winh})
}

func (img *SdlImgui) getTVColour(col uint8) imgui.PackedColor {
	c := img.cache.TV.GetFrameInfo().Spec.GetColor(signal.ColorSignal(col))
	v := imgui.Vec4{
		X: float32(c.R) / 255,
		Y: float32(c.G) / 255,
		Z: float32(c.B) / 255,
		W: float32(c.A) / 255,
	}
	return imgui.PackedColorFromVec4(v)
}
