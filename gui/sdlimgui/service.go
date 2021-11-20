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
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/userinput"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/veandco/go-sdl2/sdl"
)

// Service implements GuiCreator interface.
func (img *SdlImgui) Service() {
	// refresh lazy values
	switch img.mode {
	case emulation.ModeDebugger:
		img.lz.Refresh()
	case emulation.ModePlay:
		img.lz.FastRefresh()
	}

	// poll for sdl event or timeout
	ev := img.polling.wait()

	// some events we want to service even if an event channel has not been
	// set. very few "modes" will not set a channel but we should be mindful of
	// those events that we handle entirely within this GUI implementation
	//
	// for those events that do require an event channel to be defined we wrap
	// them in select block and log the dropped event in the default case
	for ; ev != nil; ev = sdl.PollEvent() {
		switch ev := ev.(type) {
		case *sdl.QuitEvent:
			img.quit()

		case *sdl.WindowEvent:
			img.screen.crit.section.Lock()
			img.playScr.setScaling()
			img.polling.alert()
			img.screen.crit.section.Unlock()

		case *sdl.TextInputEvent:
			if img.hasModal || !img.isCaptured() {
				img.io.AddInputCharacters(string(ev.Text[:]))
			}

		case *sdl.KeyboardEvent:
			img.serviceKeyboard(ev)

		case *sdl.MouseButtonEvent:
			// the button event to send
			var button userinput.MouseButton

			switch ev.Button {
			case sdl.BUTTON_LEFT:
				button = userinput.MouseButtonLeft

			case sdl.BUTTON_RIGHT:
				button = userinput.MouseButtonRight

				if ev.Type == sdl.MOUSEBUTTONUP {
					if img.isCaptured() {
						img.setCapture(false)
					} else if img.isPlaymode() {
						img.setCapture(true)
					}
				}
			}

			if img.isCaptured() {
				select {
				case img.userinput <- userinput.EventMouseButton{
					Button: button,
					Down:   ev.Type == sdl.MOUSEBUTTONDOWN}:
				default:
					if ev.Type == sdl.MOUSEBUTTONDOWN {
						logger.Log("sdlimgui", "dropped mouse down event")
					} else {
						logger.Log("sdlimgui", "dropped mouse up event")
					}
				}
			}

			// trigger service wake in time for next Service() iteration.
			// without this, the results of the mouse button will not be
			// seen until the timeout (in the next iteration) has elapsed.
			//
			// eg. closing a window: the window will be drawn on *this*
			// frame and *this* mouse button press will be acknowledged.
			// next frame the window will not be drawn. however, the *next*
			// frame will sleep until the time out - *this* mouse button
			// event has been consumed. calling alert() ensures there is no
			// delay in drawing the *next* frame
			img.polling.alert()

		case *sdl.MouseWheelEvent:
			var deltaX, deltaY float32
			if ev.X > 0 {
				deltaX++
			} else if ev.X < 0 {
				deltaX--
			}
			if ev.Y > 0 {
				deltaY++
			} else if ev.Y < 0 {
				deltaY--
			}
			img.io.AddMouseWheelDelta(deltaX*2, deltaY*2)

			if !img.wm.selectROM.open {
				select {
				case img.userinput <- userinput.EventMouseWheel{Delta: deltaY}:
				default:
					logger.Log("sdlimgui", "dropped mouse wheel event")
				}
			}

		case *sdl.JoyButtonEvent:
			button := userinput.GamepadButtonNone
			switch ev.Button {
			case 0:
				button = userinput.GamepadButtonA
			case 4:
				button = userinput.GamepadButtonBumperLeft
			case 5:
				button = userinput.GamepadButtonBumperRight
			case 6:
				button = userinput.GamepadButtonBack
			case 7:
				button = userinput.GamepadButtonStart
			case 8:
				button = userinput.GamepadButtonGuide
			}

			if button != userinput.GamepadButtonNone {
				select {
				case img.userinput <- userinput.EventGamepadButton{
					ID:     plugging.PortLeftPlayer,
					Button: button,
					Down:   ev.State == 1,
				}:
				default:
					logger.Log("sdlimgui", "dropped gamepad button event")
				}
			}

		case *sdl.JoyHatEvent:
			dir := userinput.DPadNone
			switch ev.Value {
			case sdl.HAT_CENTERED:
				dir = userinput.DPadCentre
			case sdl.HAT_UP:
				dir = userinput.DPadUp
			case sdl.HAT_DOWN:
				dir = userinput.DPadDown
			case sdl.HAT_LEFT:
				dir = userinput.DPadLeft
			case sdl.HAT_RIGHT:
				dir = userinput.DPadRight
			case sdl.HAT_LEFTUP:
				dir = userinput.DPadLeftUp
			case sdl.HAT_LEFTDOWN:
				dir = userinput.DPadLeftDown
			case sdl.HAT_RIGHTUP:
				dir = userinput.DPadRightUp
			case sdl.HAT_RIGHTDOWN:
				dir = userinput.DPadRightDown
			}

			if dir != userinput.DPadNone {
				select {
				case img.userinput <- userinput.EventGamepadDPad{
					ID:        plugging.PortLeftPlayer,
					Direction: dir,
				}:
				default:
					logger.Log("sdlimgui", "dropped gamepad dpad event")
				}
			}

		case *sdl.JoyAxisEvent:
			pad := sdl.GameControllerFromInstanceID(ev.Which)
			switch ev.Axis {
			case 0:
				fallthrough
			case 1:
				select {
				case img.userinput <- userinput.EventGamepadThumbstick{
					ID:         plugging.PortLeftPlayer,
					Thumbstick: userinput.GamepadThumbstickLeft,
					Horiz:      pad.Axis(0),
					Vert:       pad.Axis(1),
				}:
				default:
					logger.Log("sdlimgui", "dropped gamepad axis event")
				}
			case 3:
				fallthrough
			case 4:
				select {
				case img.userinput <- userinput.EventGamepadThumbstick{
					ID:         plugging.PortLeftPlayer,
					Thumbstick: userinput.GamepadThumbstickRight,
					Horiz:      pad.Axis(3),
					Vert:       pad.Axis(4),
				}:
				default:
					logger.Log("sdlimgui", "dropped gamepad axis event")
				}
			default:
			}

			trigger := userinput.GamepadTriggerNone
			switch ev.Axis {
			case 2:
				trigger = userinput.GamepadTriggerLeft
			case 5:
				trigger = userinput.GamepadTriggerRight
			}

			if trigger != userinput.GamepadTriggerNone {
				select {
				case img.userinput <- userinput.EventGamepadTrigger{
					ID:      plugging.PortLeftPlayer,
					Trigger: trigger,
					Amount:  ev.Value,
				}:
				default:
					logger.Log("sdlimgui", "dropped gamepad axis event")
				}
			}
		}
	}

	// mouse motion
	if img.isCaptured() {
		mx, my, _ := sdl.GetMouseState()
		if mx != img.mouseX || my != img.mouseY {
			w, h := img.plt.window.GetSize()

			// reduce mouse x and y coordintes to the range 0.0 to 1.0
			//  no need to worry about negative numbers and numbers greater
			//  than 1.0 because we (should) have restricted mouse movement
			//  to the window (with window.SetGrab(). see the ReqCaptureMouse
			//  case in the ReqFeature() function)
			x := float32(mx) / float32(w)
			y := float32(my) / float32(h)

			select {
			case img.userinput <- userinput.EventMouseMotion{
				X: x,
				Y: y,
			}:
			default:
				logger.Log("sdlimgui", "dropped mouse motion event")
			}
			img.mouseX = mx
			img.mouseY = my
		}
	}

	// start of a new frame
	img.plt.newFrame()
	imgui.NewFrame()

	// draw all windows according to debug/playmode
	img.draw()

	// rendering
	imgui.Render() // This call only creates the draw data list. Actual rendering to framebuffer is done below.
	img.glsl.preRender()
	img.screen.render()
	img.glsl.render()
	img.plt.postRender()
}

func (img *SdlImgui) serviceKeyboard(ev *sdl.KeyboardEvent) {
	if ev.Repeat == 1 {
		return
	}

	if ev.Type == sdl.KEYUP && ev.Repeat == 0 {
		handled := true

		if img.isPlaymode() {
			switch ev.Keysym.Scancode {
			case sdl.SCANCODE_ESCAPE:
				img.quit()
			case sdl.SCANCODE_F7:
				img.playScr.fpsOpen = !img.playScr.fpsOpen
			case sdl.SCANCODE_TAB:
				if !img.wm.selectROM.open {
					img.wm.selectROM.setOpen(true)
				}
			default:
				handled = false
			}
		}

		switch ev.Keysym.Scancode {
		case sdl.SCANCODE_GRAVE:
			if img.isPlaymode() {
				img.emulation.SetFeature(emulation.ReqSetMode, emulation.ModeDebugger)
			} else {
				img.emulation.SetFeature(emulation.ReqSetMode, emulation.ModePlay)
			}

		case sdl.SCANCODE_F9:
			w := img.wm.windows[winTrackerID]
			w.setOpen(!w.isOpen())

		case sdl.SCANCODE_F10:
			w := img.wm.windows[winPrefsID]
			w.setOpen(!w.isOpen())

		case sdl.SCANCODE_F11:
			img.prefs.fullScreen.Set(!img.prefs.fullScreen.Get().(bool))

		case sdl.SCANCODE_F12:
			shift := ev.Keysym.Mod&sdl.KMOD_LSHIFT == sdl.KMOD_LSHIFT || ev.Keysym.Mod&sdl.KMOD_RSHIFT == sdl.KMOD_RSHIFT
			ctrl := ev.Keysym.Mod&sdl.KMOD_LCTRL == sdl.KMOD_LCTRL || ev.Keysym.Mod&sdl.KMOD_RCTRL == sdl.KMOD_RCTRL

			if ctrl && !shift {
				img.glsl.shaders[playscrShaderID].(*playscrShader).scheduleScreenshot(modeTriple)
			} else if shift && !ctrl {
				img.glsl.shaders[playscrShaderID].(*playscrShader).scheduleScreenshot(modeDouble)
			} else {
				img.glsl.shaders[playscrShaderID].(*playscrShader).scheduleScreenshot(modeSingle)
			}

			img.playScr.emulationEvent.set(emulation.EventScreenshot)

		case sdl.SCANCODE_F14:
			fallthrough
		case sdl.SCANCODE_SCROLLLOCK:
			img.setCapture(!img.isCaptured())

		case sdl.SCANCODE_F15:
			fallthrough
		case sdl.SCANCODE_PAUSE:
			if img.isPlaymode() {
				var err error
				if img.emulation.State() == emulation.Paused {
					err = img.emulation.SetFeature(emulation.ReqSetPause, false)
				} else {
					err = img.emulation.SetFeature(emulation.ReqSetPause, true)
				}
				if err != nil {
					logger.Logf("sdlimgui", err.Error())
				}
			} else {
				if img.emulation.State() == emulation.Paused {
					img.term.pushCommand("RUN")
				} else {
					img.term.pushCommand("HALT")
				}
			}

		default:
			handled = false
		}

		if handled {
			return
		}
	}

	// forward keypresses to userinput.Event channel
	if img.isPlaymode() || img.isCaptured() {
		mod := userinput.KeyModNone

		if sdl.GetModState()&sdl.KMOD_LALT == sdl.KMOD_LALT ||
			sdl.GetModState()&sdl.KMOD_RALT == sdl.KMOD_RALT {
			mod = userinput.KeyModAlt
		} else if sdl.GetModState()&sdl.KMOD_LSHIFT == sdl.KMOD_LSHIFT ||
			sdl.GetModState()&sdl.KMOD_RSHIFT == sdl.KMOD_RSHIFT {
			mod = userinput.KeyModShift
		} else if sdl.GetModState()&sdl.KMOD_LCTRL == sdl.KMOD_LCTRL ||
			sdl.GetModState()&sdl.KMOD_RCTRL == sdl.KMOD_RCTRL {
			mod = userinput.KeyModCtrl
		}

		switch ev.Type {
		case sdl.KEYDOWN:
			fallthrough
		case sdl.KEYUP:
			select {
			case img.userinput <- userinput.EventKeyboard{
				Key:  sdl.GetScancodeName(ev.Keysym.Scancode),
				Down: ev.Type == sdl.KEYDOWN,
				Mod:  mod,
			}:
			default:
				logger.Log("sdlimgui", "dropped key up event")
			}
		}

		return
	}

	// remaining keypresses forwarded to imgui io system
	switch ev.Type {
	case sdl.KEYDOWN:
		img.io.KeyPress(int(ev.Keysym.Scancode))
		img.updateKeyModifier()
	case sdl.KEYUP:
		img.io.KeyRelease(int(ev.Keysym.Scancode))
		img.updateKeyModifier()
	}
}
