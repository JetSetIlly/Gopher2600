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

	// a short delay after a window event seems to help the window to resync
	// with the monitor's refresh rate
	resync      int
	windowEvent bool

	// ideal frame time in nanoseconds
	frameDuration time.Duration

	// the start time of the last frame taken from just after the
	renderStart     time.Time
	renderAvgTime   time.Duration
	renderAvgTimeCt int
	renderAlert     bool
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

	// set hints for application
	sdl.SetHint(sdl.HINT_VIDEO_MINIMIZE_ON_FOCUS_LOSS, "0")
	sdl.SetHint(sdl.HINT_VIDEO_X11_NET_WM_BYPASS_COMPOSITOR, "0")
	sdl.SetHint(sdl.HINT_VIDEO_X11_NET_WM_PING, "0")

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

	// log SDL information
	var sdlVersion sdl.Version
	sdl.VERSION(&sdlVersion)
	logger.Logf(logger.Allow, "sdl", "version %d.%d.%d", sdlVersion.Major, sdlVersion.Minor, sdlVersion.Patch)
	logger.Logf(logger.Allow, "sdl", "using GL version %d.%d%s", major, minor, profile_s)

	plt := &platform{
		img: img,
	}

	plt.mode, err = sdl.GetCurrentDisplayMode(0)
	if err != nil {
		sdl.Quit()
		return nil, fmt.Errorf("sdl: %w", err)
	}
	logger.Logf(logger.Allow, "sdl", "refresh rate: %dHz", plt.mode.RefreshRate)

	var title string
	ver, rev, rel := version.Version()
	if rel {
		title = fmt.Sprintf("%s (%s)", version.ApplicationName, ver)
	} else {
		title = fmt.Sprintf("%s (%s)", version.ApplicationName, rev)
	}
	plt.window, err = sdl.CreateWindow(title,
		sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		int32(float32(plt.mode.W)*0.80), int32(float32(plt.mode.H)*0.80),
		sdl.WINDOW_OPENGL|sdl.WINDOW_ALLOW_HIGHDPI|sdl.WINDOW_RESIZABLE|sdl.WINDOW_HIDDEN)

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
			logger.Logf(logger.Allow, "sdl", "gamepad: %s", pad.Joystick().Name())
			plt.joysticks = append(plt.joysticks, controller{closeController: pad})
		} else {
			joy := sdl.JoystickOpen(i)
			if joy.Attached() {
				logger.Logf(logger.Allow, "sdl", "joystick: %s", joy.Name())
				plt.joysticks = append(plt.joysticks, controller{
					closeController: joy,
					isStelladaptor:  strings.Contains(strings.ToLower(joy.Name()), "stelladaptor"),
				})
			}
		}
	}

	if len(plt.joysticks) == 0 {
		logger.Log(logger.Allow, "sdl", "no joysticks/gamepads found")
	}

	// duration of each frame according to monitor refresh rate
	plt.frameDuration = time.Duration(1000000000/int64(plt.mode.RefreshRate)) * time.Nanosecond

	// calculate the average render time every second
	plt.renderStart = time.Now()

	return plt, nil
}

// call finalise after initialising imgui (which must happen after SDL creation
// in the newPlatform() function)
func (plt *platform) finalisePlatform() error {
	return nil
}

// list of swap intervalue values
const (
	syncImmediateUpdate     = 0
	syncWithVerticalRetrace = 1
	syncAdaptive            = -1
)

func (plt *platform) setSwapInterval(i int) {
	err := sdl.GLSetSwapInterval(i)
	if err != nil {
		logger.Logf(logger.Allow, "sdl", "GLSetSwapInterval(%d): %v", i, err)
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

// windowSize returns the dimension of the display.
func (plt *platform) windowSize() (width, height float32) {
	w, h := plt.window.GetSize()
	return float32(w), float32(h)
}

// displayDPI returns the dots/inch for the display the window is in
func (plt *platform) displayDPI() (float32, error) {
	ww, _ := plt.window.GetSize()
	dw, _ := plt.window.GLGetDrawableSize()
	dpiScale := float32(ww) / float32(dw)

	_, dpi, _, err := sdl.GetDisplayDPI(0)
	return dpi * dpiScale, err
}

// framebufferSize returns the dimension of the framebuffer.
func (plt *platform) framebufferSize() (width, height float32) {
	w, h := plt.window.GLGetDrawableSize()
	return float32(w), float32(h)
}

// newFrame marks the begin of a render pass. It forwards all current state to imgui.CurrentIO().
func (plt *platform) newFrame() {
	// Setup display size (every frame to accommodate for window resizing)
	winw, winh := plt.windowSize()
	imgui.CurrentIO().SetDisplaySize(imgui.Vec2{X: winw, Y: winh})

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
	timeDiff := time.Since(plt.renderStart)
	defer func() {
		plt.renderStart = time.Now()
	}()
	if plt.renderAvgTimeCt > 1 {
		plt.renderAvgTime = plt.renderAvgTime + (timeDiff-plt.renderAvgTime)/time.Duration(min(plt.renderAvgTimeCt, 120))
		if plt.renderAvgTimeCt > 120 {
			plt.renderAvgTimeCt = 1
		}
	}
	plt.renderAvgTimeCt++

	// alert if frame has taken a long time to render
	if timeDiff > plt.frameDuration {
		plt.renderAlert = true
	} else {
		plt.renderAlert = false
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
}

// set the capture state for the mouse
func (plt *platform) setCapture(set bool) {
	sdl.SetRelativeMouseMode(set)
}
