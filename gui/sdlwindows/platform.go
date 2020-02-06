package sdlwindows

import (
	"fmt"
	"gopher2600/test"
	"runtime"

	"github.com/inkyblackness/imgui-go/v2"
	"github.com/veandco/go-sdl2/sdl"
)

const windowTitle = "Gopher2600"
const windowTitleCaptured = "Gopher2600 [captured]"

type platform struct {
	wnd *SdlWindows

	window     *sdl.Window
	shouldStop bool

	time        uint64
	buttonsDown [3]bool
}

// newPlatform is the preferred method of initialisation for the platform type
func newPlatform(wnd *SdlWindows) (*platform, error) {
	runtime.LockOSThread()

	err := sdl.Init(sdl.INIT_VIDEO)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SDL2: %v", err)
	}

	plt := &platform{
		wnd: wnd,
	}

	plt.window, err = sdl.CreateWindow(windowTitle,
		sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED,
		1280, 720,
		sdl.WINDOW_OPENGL|sdl.WINDOW_HIDDEN)

	if err != nil {
		sdl.Quit()
		return nil, fmt.Errorf("failed to create window: %v", err)
	}

	_ = sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 2)
	_ = sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 1)
	_ = sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 3)
	_ = sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 2)
	_ = sdl.GLSetAttribute(sdl.GL_CONTEXT_FLAGS, sdl.GL_CONTEXT_FORWARD_COMPATIBLE_FLAG)
	_ = sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)
	_ = sdl.GLSetAttribute(sdl.GL_DOUBLEBUFFER, 1)
	_ = sdl.GLSetAttribute(sdl.GL_DEPTH_SIZE, 24)
	_ = sdl.GLSetAttribute(sdl.GL_STENCIL_SIZE, 8)

	glContext, err := plt.window.GLCreateContext()
	if err != nil {
		plt.destroy()
		return nil, fmt.Errorf("failed to create OpenGL context: %v", err)
	}
	err = plt.window.GLMakeCurrent(glContext)
	if err != nil {
		plt.destroy()
		return nil, fmt.Errorf("failed to set current OpenGL context: %v", err)
	}

	_ = sdl.GLSetSwapInterval(1)

	return plt, nil
}

// destroy cleans up the resources.
func (plt *platform) destroy() {
	if plt.window != nil {
		_ = plt.window.Destroy()
		plt.window = nil
	}
	sdl.Quit()
}

// setDisplaySize resizes the window
func (plt *platform) setDisplaySize(w, h int) {
	plt.window.SetSize(int32(w), int32(h))
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
	plt.wnd.io.SetDisplaySize(imgui.Vec2{X: displaySize[0], Y: displaySize[1]})

	// Setup time step (we don't use SDL_GetTicks() because it is using millisecond resolution)
	frequency := sdl.GetPerformanceFrequency()
	currentTime := sdl.GetPerformanceCounter()
	if plt.time > 0 {
		plt.wnd.io.SetDeltaTime(float32(currentTime-plt.time) / float32(frequency))
	} else {
		plt.wnd.io.SetDeltaTime(1.0 / 60.0)
	}
	plt.time = currentTime

	// If a mouse press event came, always pass it as "mouse held this frame", so we don't miss click-release events that are shorter than 1 frame.
	x, y, state := sdl.GetMouseState()
	plt.wnd.io.SetMousePosition(imgui.Vec2{X: float32(x), Y: float32(y)})
	for i, button := range []uint32{sdl.BUTTON_LEFT, sdl.BUTTON_RIGHT, sdl.BUTTON_MIDDLE} {
		plt.wnd.io.SetMouseButtonDown(i, plt.buttonsDown[i] || (state&sdl.Button(button)) != 0)
		plt.buttonsDown[i] = false
	}
}

// PostRender performs a buffer swap.
func (plt *platform) postRender() {
	plt.window.GLSwap()
}

// show the main window (or not)
//
// MUST NOT be called from the #mainthread
func (plt *platform) showWindow(show bool) {
	test.AssertNonMainThread()

	plt.wnd.service <- func() {
		if show {
			plt.window.Show()
		} else {
			plt.window.Hide()
		}
	}
}
