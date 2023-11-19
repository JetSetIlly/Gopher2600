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
	"runtime"
	"strings"
	"time"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/version"
	"github.com/veandco/go-sdl2/sdl"
)

const windowTitle = "Gopher2600"

// the closeController interface is implemented by SDL joysticks and gamepads. From our point
// of view we only need to close the closeController when we are done with it
type closeController interface {
	Close()
}

type controller struct {
	closeController
	isStelladaptor bool
}

// global control of gamepad support
const supportGamepads = true

type platform struct {
	img    *SdlImgui
	window *sdl.Window
	mode   sdl.DisplayMode

	joysticks []controller

	// trickle mouse buttons
	trickleMouseButtonLeft  trickleMouseButton
	trickleMouseButtonRight trickleMouseButton

	// use ticker to synchronise with monitor
	syncTicker *time.Ticker
}

// trickle mouse button is a mechanism that allows a mouse button down/up event
// that occurs in the same frame to be serviced by the dear imgui io system
//
// as of dear imgui version 1.87 this has been solved with the AddMouseButtonEvent()
// function. we're not currently use version of dear imgui but we should
// consider replacing this trickleMouseButton type if we ever move to the new
// version
//
// the mechanism was added to mitigate a problem with Apple "touchpads" that
// simulate mouse presses simply through touch (as opposed to clicking)
type trickleMouseButton int

// list of valid trickleMouseButton values
const (
	trickleMouseNone trickleMouseButton = 0
	trickleMouseUp   trickleMouseButton = 1
	trickleMouseDown trickleMouseButton = 2
)

// newPlatform is the preferred method of initialisation for the platform type.
func newPlatform(img *SdlImgui) (*platform, error) {
	// the SDL package calls LockOSThread() but we call it here too. it can't
	// hurt and we never unlock it in any case
	runtime.LockOSThread()

	err := sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		return nil, fmt.Errorf("sdl: %w", err)
	}

	switch img.rnd.requires() {
	case requiresOpenGL32:
		err = sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 3)
		if err != nil {
			return nil, fmt.Errorf("sdl: %w", err)
		}
		err = sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 2)
		if err != nil {
			return nil, fmt.Errorf("sdl: %w", err)
		}
		err = sdl.GLSetAttribute(sdl.GL_CONTEXT_FLAGS, sdl.GL_CONTEXT_FORWARD_COMPATIBLE_FLAG)
		if err != nil {
			return nil, fmt.Errorf("sdl: %w", err)
		}
		err = sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)
		if err != nil {
			return nil, fmt.Errorf("sdl: %w", err)
		}
	case requiresOpenGL21:
		err = sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 2)
		if err != nil {
			return nil, fmt.Errorf("sdl: %w", err)
		}
		err = sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 1)
		if err != nil {
			return nil, fmt.Errorf("sdl: %w", err)
		}
	}

	major, err := sdl.GLGetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION)
	if err != nil {
		return nil, fmt.Errorf("sdl: %w", err)
	}
	minor, err := sdl.GLGetAttribute(sdl.GL_CONTEXT_MINOR_VERSION)
	if err != nil {
		return nil, fmt.Errorf("sdl: %w", err)
	}
	profile, err := sdl.GLGetAttribute(sdl.GL_CONTEXT_PROFILE_MASK)
	if err != nil {
		return nil, fmt.Errorf("sdl: %w", err)
	}
	var profile_s string
	switch profile {
	case sdl.GL_CONTEXT_PROFILE_CORE:
		profile_s = " core"
	case sdl.GL_CONTEXT_PROFILE_COMPATIBILITY:
		profile_s = " compatibility"
	case sdl.GL_CONTEXT_PROFILE_ES:
		profile_s = " ES"
	}

	var sdlVersion sdl.Version
	sdl.VERSION(&sdlVersion)
	logger.Logf("sdl", "version %d.%d.%d", sdlVersion.Major, sdlVersion.Minor, sdlVersion.Patch)
	logger.Logf("sdl", "using GL version %d.%d%s", major, minor, profile_s)

	plt := &platform{
		img: img,
	}

	plt.mode, err = sdl.GetCurrentDisplayMode(0)
	if err != nil {
		sdl.Quit()
		return nil, fmt.Errorf("sdl: %w", err)
	}
	logger.Logf("sdl", "refresh rate: %dHz", plt.mode.RefreshRate)

	// map sdl key codes to imgui codes
	plt.setKeyMapping()

	plt.window, err = sdl.CreateWindow(fmt.Sprintf("%s (%s)", windowTitle, version.Version),
		sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		int32(float32(plt.mode.W)*0.80), int32(float32(plt.mode.H)*0.80),
		sdl.WINDOW_OPENGL|sdl.WINDOW_ALLOW_HIGHDPI|sdl.WINDOW_RESIZABLE|sdl.WINDOW_HIDDEN|sdl.WINDOW_FOREIGN)

	if err != nil {
		sdl.Quit()
		return nil, fmt.Errorf("sdl: %w", err)
	}

	glContext, err := plt.window.GLCreateContext()
	if err != nil {
		_ = plt.destroy()
		return nil, fmt.Errorf("sdl: %w", err)
	}
	err = plt.window.GLMakeCurrent(glContext)
	if err != nil {
		_ = plt.destroy()
		return nil, fmt.Errorf("sdl: %w", err)
	}

	// add joysticks
	for i := 0; i < sdl.NumJoysticks(); i++ {
		var pad *sdl.GameController
		if supportGamepads {
			pad = sdl.GameControllerOpen(i)
		}
		if supportGamepads && pad.Attached() {
			logger.Logf("sdl", "gamepad: %s", pad.Joystick().Name())
			plt.joysticks = append(plt.joysticks, controller{closeController: pad})
		} else {
			joy := sdl.JoystickOpen(i)
			if joy.Attached() {
				logger.Logf("sdl", "joystick: %s", joy.Name())
				plt.joysticks = append(plt.joysticks, controller{
					closeController: joy,
					isStelladaptor:  strings.Contains(strings.ToLower(joy.Name()), "stelladaptor"),
				})
			}
		}
	}

	if len(plt.joysticks) == 0 {
		logger.Log("sdl", "no joysticks/gamepads found")
	}

	return plt, nil
}

// list of swap intervalue values. with the exception of syncTicker all of these
// are values defined and expected by the SDL.GLSetSwapInterval() function
const (
	syncImmediateUpdate     = 0
	syncWithVerticalRetrace = 1
	syncAdaptive            = -1
	syncTicker              = 2
)

func (plt *platform) setSwapInterval(i int) {
	if i == syncTicker {
		// ticker to control update frequency
		d := time.Duration(1000000000/int64(plt.mode.RefreshRate)) * time.Nanosecond
		plt.syncTicker = time.NewTicker(d)

		// in reality syncTicker requires us to set GL swap interval to 0
		i = 0
	} else {
		if plt.syncTicker != nil {
			plt.syncTicker.Stop()
		}
		plt.syncTicker = nil
	}

	err := sdl.GLSetSwapInterval(i)
	if err != nil {
		logger.Logf("sdl", "GLSetSwapInterval(%d): %s", i, err.Error())
	}
}

// destroy cleans up the resources.
func (plt *platform) destroy() error {
	for _, joy := range plt.joysticks {
		joy.Close()
	}

	if plt.window != nil {
		err := plt.window.Destroy()
		if err != nil {
			return err
		}
		plt.window = nil
	}
	sdl.Quit()

	return nil
}

// displaySize returns the dimension of the display.
func (plt *platform) displaySize() [2]float32 {
	w, h := plt.window.GetSize()
	return [2]float32{float32(w), float32(h)}
}

// framebufferSize returns the dimension of the framebuffer.
func (plt *platform) framebufferSize() [2]float32 {
	w, h := plt.window.GLGetDrawableSize()
	return [2]float32{float32(w), float32(h)}
}

// newFrame marks the begin of a render pass. It forwards all current state to imgui.CurrentIO().
func (plt *platform) newFrame() {
	// Setup display size (every frame to accommodate for window resizing)
	displaySize := plt.displaySize()
	imgui.CurrentIO().SetDisplaySize(imgui.Vec2{X: displaySize[0], Y: displaySize[1]})

	// if mouse is captured then do not update imgui mouse information.
	if !plt.img.isCaptured() {
		// If a mouse press event came, always pass it as "mouse held this frame",
		// so we don't miss click-release events that are shorter than 1 frame.
		x, y, state := sdl.GetMouseState()

		imgui.CurrentIO().SetMousePosition(imgui.Vec2{X: float32(x), Y: float32(y)})
		for i, button := range []uint32{sdl.BUTTON_LEFT, sdl.BUTTON_RIGHT, sdl.BUTTON_MIDDLE} {
			imgui.CurrentIO().SetMouseButtonDown(i, (state&sdl.Button(button)) != 0)
		}

		// trickle event handling will supercede any previous SetMouseButtonDown() calls

		switch plt.trickleMouseButtonLeft {
		case trickleMouseDown:
			imgui.CurrentIO().SetMouseButtonDown(0, true)
			plt.trickleMouseButtonLeft = trickleMouseUp
		case trickleMouseUp:
			imgui.CurrentIO().SetMouseButtonDown(0, false)
			plt.trickleMouseButtonLeft = trickleMouseNone
		case trickleMouseNone:
		}

		switch plt.trickleMouseButtonRight {
		case trickleMouseDown:
			imgui.CurrentIO().SetMouseButtonDown(1, true)
			plt.trickleMouseButtonRight = trickleMouseUp
		case trickleMouseUp:
			imgui.CurrentIO().SetMouseButtonDown(1, false)
			plt.trickleMouseButtonRight = trickleMouseNone
		case trickleMouseNone:
		}
	}
}

// PostRender performs a buffer swap.
func (plt *platform) postRender() {
	if plt.syncTicker != nil {
		<-plt.syncTicker.C
	}
	plt.window.GLSwap()
}

// toggle the full screeens state. does not capture mouse.
func (plt *platform) setFullScreen(fullScreen bool) {
	if fullScreen {
		plt.window.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
	} else {
		plt.window.SetFullscreen(0)
	}

	// a short delay seems to smooth things out by giving time for the system
	// to make the changes to the full screen state
	<-time.After(100 * time.Millisecond)
}
