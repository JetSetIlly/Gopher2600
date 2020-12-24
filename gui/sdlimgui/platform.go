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

	"github.com/inkyblackness/imgui-go/v2"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/veandco/go-sdl2/sdl"
)

const windowTitle = "Gopher2600"

type platform struct {
	img    *SdlImgui
	window *sdl.Window
	mode   sdl.DisplayMode

	// whether window is full screen or not
	fullScreen bool

	// event returned by WaitEvent(). used to listen to events in a select
	// block along with other channels
	event chan sdl.Event
}

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
	logger.Log("sdl", fmt.Sprintf("refresh rate: %dHz", plt.mode.RefreshRate))

	// map sdl key codes to imgui codes
	plt.setKeyMapping()

	plt.window, err = sdl.CreateWindow(windowTitle,
		sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		1280, 740,
		sdl.WINDOW_OPENGL|sdl.WINDOW_ALLOW_HIGHDPI|sdl.WINDOW_RESIZABLE|sdl.WINDOW_HIDDEN)

	if err != nil {
		sdl.Quit()
		return nil, fmt.Errorf("sdl: %v", err)
	}

	glContext, err := plt.window.GLCreateContext()
	if err != nil {
		err = plt.destroy()
		return nil, fmt.Errorf("sdl: %v", err)
	}
	err = plt.window.GLMakeCurrent(glContext)
	if err != nil {
		err = plt.destroy()
		return nil, fmt.Errorf("sdl: %v", err)
	}

	if sdl.GLSetSwapInterval(1) != nil {
		logger.Log("sdl", "cannot set GLSwapInterval() for SDL GUI")
	}

	plt.event = make(chan sdl.Event, 1)
	go func() {
		for {
			plt.event <- sdl.WaitEvent()
		}
	}()

	return plt, nil
}

// destroy cleans up the resources.
func (plt *platform) destroy() error {
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

	// If a mouse press event came, always pass it as "mouse held this frame",
	// so we don't miss click-release events that are shorter than 1 frame.
	x, y, state := sdl.GetMouseState()

	// if mouse is captured and the mouse is not over the tv screen then ignore
	// the mouse button state. the check against isHovered because we want
	// imgui to recognise the initial click to activate the window. the check
	// against isCaptured is because we don't want the tv screen to be
	// deactivated when the "invisible" mouse is outside the tv screen bounds.
	//
	// TODO: roll mouse updates into service loop
	if plt.img.isCaptured() && !plt.img.wm.dbgScr.isHovered {
		state = 0
	}

	plt.img.io.SetMousePosition(imgui.Vec2{X: float32(x), Y: float32(y)})
	for i, button := range []uint32{sdl.BUTTON_LEFT, sdl.BUTTON_RIGHT, sdl.BUTTON_MIDDLE} {
		plt.img.io.SetMouseButtonDown(i, (state&sdl.Button(button)) != 0)
	}
}

// PostRender performs a buffer swap.
func (plt *platform) postRender() {
	plt.window.GLSwap()
}

// toggle the full screeens state.
func (plt *platform) toggleFullScreen() {
	plt.fullScreen = !plt.fullScreen
	if plt.fullScreen {
		plt.window.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
		plt.img.setCapture(true)
	} else {
		plt.window.SetFullscreen(0)
		plt.img.setCapture(false)
	}
}
