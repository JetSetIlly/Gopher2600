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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package sdlwindows

import (
	"gopher2600/gui"
	"gopher2600/paths"
	"io"
	"time"

	"github.com/inkyblackness/imgui-go/v2"
)

// SdlWindows is a fully featured windowed debugger
//
// MUST ONLY be called from the #mainthread
type SdlWindows struct {
	io       imgui.IO
	context  *imgui.Context
	platform *platform
	renderer *renderer

	// events channel is not created but assigned with SetEventChannel()
	events chan gui.Event
}

// NewSdlWindows is the preferred method of initialisation for type SdlWindows
func NewSdlWindows() (*SdlWindows, error) {
	wnd := &SdlWindows{
		context: imgui.CreateContext(nil),
		io:      imgui.CurrentIO(),
	}

	var err error

	wnd.platform, err = newPlatform(wnd.io)
	if err != nil {
		return nil, err
	}

	wnd.renderer, err = newRenderer(wnd.io)
	if err != nil {
		return nil, err
	}

	iniPath, err := paths.ResourcePath("", "imgui.ini")
	if err != nil {
		return nil, err
	}
	wnd.io.SetIniFilename(iniPath)

	return wnd, nil
}

// Destroy implements GuiCreator interface
//
// MUST ONLY be called from the #mainthread
func (wnd *SdlWindows) Destroy(output io.Writer) {
	wnd.renderer.destroy()
	wnd.platform.destroy()
	wnd.context.Destroy()
}

// IsVisible implements gui.GUI interface
func (wnd *SdlWindows) IsVisible() bool {
	return true
}

// SetFeature implements gui.GUI interface
func (wnd *SdlWindows) SetFeature(request gui.FeatureReq, args ...interface{}) error {
	return nil
}

// SetEventChannel implements gui.GUI interface
func (wnd *SdlWindows) SetEventChannel(events chan gui.Event) {
	wnd.events = events
}

// Service implements GuiCreator interface
func (wnd *SdlWindows) Service() {
	wnd.platform.processEvents()
	if wnd.platform.shouldStop {
		wnd.events <- gui.EventWindowClose{}
	}

	// Signal start of a new frame
	wnd.platform.newFrame()
	imgui.NewFrame()

	// imgui commands
	imgui.Begin("gopher2600")
	wnd.drawWindows()
	imgui.End()

	// Rendering
	imgui.Render() // This call only creates the draw data list. Actual rendering to framebuffer is done below.

	clearColor := [4]float32{0.0, 0.0, 0.0, 1.0}
	wnd.renderer.preRender(clearColor)
	// A this point, the application could perform its own rendering...
	// app.RenderScene()

	wnd.renderer.render(wnd.platform.displaySize(), wnd.platform.framebufferSize(), imgui.RenderedDrawData())
	wnd.platform.postRender()

	// sleep to avoid 100% CPU usage for this demo
	<-time.After(time.Millisecond * 25)
}

func (wnd *SdlWindows) drawWindows() {
	imgui.Text("Hello from another window!")
	if imgui.Button("Close Me") {
		wnd.events <- gui.EventWindowClose{}
	}
}
