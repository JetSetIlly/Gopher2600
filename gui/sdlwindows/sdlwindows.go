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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlwindows

import (
	"gopher2600/gui"
	"gopher2600/paths"
	"gopher2600/performance/limiter"
	"gopher2600/television"
	"io"

	"github.com/inkyblackness/imgui-go/v2"
)

// SdlWindows is a fully featured windowed debugger
type SdlWindows struct {
	io       imgui.IO
	context  *imgui.Context
	platform *platform
	glsl     *glsl

	tv     television.Television
	screen *tvScreen

	// functions that need to be performed in the main thread should be queued
	// for service.
	service    chan func()
	serviceErr chan error

	// limit number of frames per second
	lmtr *limiter.FpsLimiter

	// events channel is not created but assigned with SetEventChannel()
	events chan gui.Event

	// window opening is delayed until television frame is stable
	showOnNextStable bool

	// mouse coords at last frame
	mx, my int32

	// whether mouse is captured
	isCaptured bool
}

// NewSdlWindows is the preferred method of initialisation for type SdlWindows
//
// MUST ONLY be called from the #mainthread
func NewSdlWindows(tv television.Television) (*SdlWindows, error) {
	wnd := &SdlWindows{
		context:    imgui.CreateContext(nil),
		io:         imgui.CurrentIO(),
		tv:         tv,
		service:    make(chan func(), 1),
		serviceErr: make(chan error, 1),
	}

	// create new frame limiter. we change the rate in the resize function
	// (rate may change due to specification change)
	wnd.lmtr = limiter.NewFPSLimiter(-1)

	var err error

	wnd.platform, err = newPlatform(wnd)
	if err != nil {
		return nil, err
	}

	wnd.glsl, err = newGlsl(wnd.io)
	if err != nil {
		return nil, err
	}

	iniPath, err := paths.ResourcePath("", "imgui.ini")
	if err != nil {
		return nil, err
	}
	wnd.io.SetIniFilename(iniPath)

	wnd.screen, err = newTvScreen(wnd)
	if err != nil {
		return nil, err
	}
	wnd.glsl.tvScreenTexture = wnd.screen.texture

	tv.AddPixelRenderer(wnd.screen)

	return wnd, nil
}

// Destroy implements GuiCreator interface
//
// MUST ONLY be called from the #mainthread
func (wnd *SdlWindows) Destroy(output io.Writer) {
	wnd.screen.destroy()
	wnd.glsl.destroy()
	wnd.platform.destroy()
	wnd.context.Destroy()
}

// SetEventChannel implements gui.GUI interface
func (wnd *SdlWindows) SetEventChannel(events chan gui.Event) {
	wnd.events = events
}
