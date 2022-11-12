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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/veandco/go-sdl2/sdl"
)

const windowTitle = "Gopher2600"

const maxGamepads = 10

type platform struct {
	img    *SdlImgui
	window *sdl.Window
	mode   sdl.DisplayMode

	gamepad []*sdl.GameController

	// trickle left mouse button
	trickleLeftMouseButton trickleMouseButton
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
	runtime.LockOSThread()

	err := sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		return nil, fmt.Errorf("sdl: %v", err)
	}

	_ = sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 3)
	_ = sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 2)
	_ = sdl.GLSetAttribute(sdl.GL_CONTEXT_FLAGS, sdl.GL_CONTEXT_FORWARD_COMPATIBLE_FLAG)
	_ = sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)
	_ = sdl.GLSetAttribute(sdl.GL_DOUBLEBUFFER, 1)
	_ = sdl.GLSetAttribute(sdl.GL_DEPTH_SIZE, 24)
	_ = sdl.GLSetAttribute(sdl.GL_STENCIL_SIZE, 8)

	plt := &platform{
		img: img,
	}

	plt.mode, err = sdl.GetCurrentDisplayMode(0)
	if err != nil {
		sdl.Quit()
		return nil, fmt.Errorf("sdl: %v", err)
	}
	logger.Logf("sdl", "refresh rate: %dHz", plt.mode.RefreshRate)

	// map sdl key codes to imgui codes
	plt.setKeyMapping()

	plt.window, err = sdl.CreateWindow(windowTitle,
		sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		int32(float32(plt.mode.W)*0.80), int32(float32(plt.mode.H)*0.80),
		sdl.WINDOW_OPENGL|sdl.WINDOW_ALLOW_HIGHDPI|sdl.WINDOW_RESIZABLE|sdl.WINDOW_HIDDEN)

	if err != nil {
		sdl.Quit()
		return nil, fmt.Errorf("sdl: %v", err)
	}

	glContext, err := plt.window.GLCreateContext()
	if err != nil {
		_ = plt.destroy()
		return nil, fmt.Errorf("sdl: %v", err)
	}
	err = plt.window.GLMakeCurrent(glContext)
	if err != nil {
		_ = plt.destroy()
		return nil, fmt.Errorf("sdl: %v", err)
	}

	// default to disabled vsync
	plt.glSetSwapInterval(0)

	// open all available gamepads
	plt.gamepad = make([]*sdl.GameController, 0, maxGamepads)

	for i := 0; i < maxGamepads; i++ {
		pad := sdl.GameControllerOpen(i)
		if pad.Attached() {
			logger.Logf("sdl", "gamepad: %s", pad.Name())
			plt.gamepad = append(plt.gamepad, pad)
		}
	}

	if len(plt.gamepad) == 0 {
		logger.Log("sdl", "no gamepad found")
	}

	return plt, nil
}

func (plt *platform) glSetSwapInterval(i int) {
	if sdl.GLSetSwapInterval(i) != nil {
		logger.Log("sdl", "cannot set GLSwapInterval() for SDL GUI")
	}
}

// destroy cleans up the resources.
func (plt *platform) destroy() error {
	for _, pad := range plt.gamepad {
		pad.Close()
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
	plt.img.io.SetDisplaySize(imgui.Vec2{X: displaySize[0], Y: displaySize[1]})

	// if mouse is captured then do not update imgui mouse information.
	if !plt.img.isCaptured() {
		// If a mouse press event came, always pass it as "mouse held this frame",
		// so we don't miss click-release events that are shorter than 1 frame.
		x, y, state := sdl.GetMouseState()

		plt.img.io.SetMousePosition(imgui.Vec2{X: float32(x), Y: float32(y)})
		for i, button := range []uint32{sdl.BUTTON_LEFT, sdl.BUTTON_RIGHT, sdl.BUTTON_MIDDLE} {
			plt.img.io.SetMouseButtonDown(i, (state&sdl.Button(button)) != 0)
		}

		// trickle event supercedes previous SetMouseButtonDown() call
		switch plt.trickleLeftMouseButton {
		case trickleMouseDown:
			plt.img.io.SetMouseButtonDown(0, true)
			plt.trickleLeftMouseButton = trickleMouseUp
		case trickleMouseUp:
			plt.img.io.SetMouseButtonDown(0, false)
			plt.trickleLeftMouseButton = trickleMouseNone
		case trickleMouseNone:
		}
	}
}

// PostRender performs a buffer swap.
func (plt *platform) postRender() {
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
