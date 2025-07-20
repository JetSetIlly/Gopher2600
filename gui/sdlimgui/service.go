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
	"time"

	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/userinput"

	"github.com/jetsetilly/imgui-go/v5"
	"github.com/veandco/go-sdl2/sdl"
)

// Service implements GuiCreator interface.
func (img *SdlImgui) Service() {
	defer func() {
		img.renderFrame()
		time.Sleep(img.polling.timeout())
	}()

	img.polling.serviceRequests()

	// handle font reset procedure
	if img.resetFonts >= 1 {
		if img.resetFonts == 1 {
			err := img.fonts.loadFonts(img.plt, img.rnd, img.prefs)
			if err != nil {
				panic(err)
			}
			img.wm.hasInitialised = false
		}
		img.resetFonts--
	}

	// phantom input system must be reset before anything else is processed
	img.phantomInput = phantomInputNone

	// whether mouse button down event have been polled. if it has and we poll
	// an up event in the same PollEvent() loop below, then we need to
	// "trickle" the up and down events over two frames. see commentary for
	// trickleMouseButton type
	var leftMouseDownPolled bool
	var rightMouseDownPolled bool

	// we only want to send one mouse event to the emulation per frame. anything
	// more is wasteful and expensive
	var mouseMotion bool
	var mouseMotionX float64
	var mouseMotionY float64

	// get user input channel from debugger
	input := img.dbg.UserInput()

	// loop until polling is false
	polling := true
	for polling {
		pev := sdl.PollEvent()
		if pev == nil {
			break // polling loop
		}

		// make sure polling is awake. this makes sure the correct timeout is to
		// be returned next frame. using a defer so that we only do this once
		// per frame
		defer func() {
			img.polling.awaken()
		}()

		switch ev := pev.(type) {
		case *sdl.QuitEvent:
			img.quit()

			// end polling immediately
			polling = false

		case *sdl.WindowEvent:
			// window events are mainly handled by serviceWindowEvent()
			// via an event filter

		case *sdl.TextInputEvent:
			if !img.modalActive() || !img.isCaptured() {
				imgui.CurrentIO().AddInputCharacters(string(ev.Text[:]))

				// text input events are perfect for indicating the
				// addition of a phantom rune. backspaces are handled in
				// the serviceKeyboard() function
				img.phantomInput = phantomInputRune
				img.phantomInputRune = rune(ev.Text[0])
			}

		case *sdl.KeyboardEvent:
			img.smartCursorVisibility(true)
			img.serviceKeyboard(ev)

		case *sdl.MouseMotionEvent:
			img.smartCursorVisibility(false)
			if img.isCaptured() {
				mouseMotion = true

				// use the most recent value sent by the mouse. accumulating the
				// relative values over the course of the frame is not suitable
				// for paddle operation
				mouseMotionX = float64(ev.XRel)
				mouseMotionY = float64(ev.YRel)
			}

		case *sdl.MouseButtonEvent:
			img.smartCursorVisibility(false)

			// the button event to send
			var button userinput.MouseButton

			switch ev.Button {
			case sdl.BUTTON_LEFT:
				button = userinput.MouseButtonLeft
				switch ev.Type {
				case sdl.MOUSEBUTTONDOWN:
					leftMouseDownPolled = true
				case sdl.MOUSEBUTTONUP:
					if leftMouseDownPolled {
						img.plt.trickleMouseButtonLeft = trickleMouseDown
					}
				}

			case sdl.BUTTON_MIDDLE:
				if img.isCaptured() {
					img.setCapture(false)
				}

			case sdl.BUTTON_RIGHT:
				button = userinput.MouseButtonRight
				switch ev.Type {
				case sdl.MOUSEBUTTONDOWN:
					rightMouseDownPolled = true
				case sdl.MOUSEBUTTONUP:
					if rightMouseDownPolled {
						img.plt.trickleMouseButtonRight = trickleMouseDown
					}

					// handling of mouse capture/release is done outside of the outside of trickle
					// mouse polling this means that capturing the mouse requires a physical click
					// on the macintosh touchpad. I think this is okay but if it's not, the
					// following code will need to be called during trickle resolution (probably
					// just a function pointer)
					if img.isCaptured() {
						if !img.isPlaymode() {
							img.setCapturedRunning(false)
						} else {
							img.setCapture(false)
						}
					} else if img.isPlaymode() {
						// set mouse capture if mouse is not over a window
						if !img.wm.playmodeCaptureInhibit {
							img.setCapture(true)
						}
					}
				}
			}

			if img.isCaptured() {
				select {
				case input <- userinput.EventMouseButton{
					Button: button,
					Down:   ev.Type == sdl.MOUSEBUTTONDOWN}:
				default:
					if ev.Type == sdl.MOUSEBUTTONDOWN {
						logger.Log(logger.Allow, "sdlimgui", "dropped mouse down event")
					} else {
						logger.Log(logger.Allow, "sdlimgui", "dropped mouse up event")
					}
				}
			}

		case *sdl.MouseWheelEvent:
			// only respond to mouse wheel events if the window has
			// input focus. this is because without input focus
			// getKeyMod() will always return userinput.KeyModNone. it
			// is confusing if the mousewheel is working but no keyboard
			// modifiers are affecting it
			if img.plt.window.GetFlags()&sdl.WINDOW_INPUT_FOCUS == sdl.WINDOW_INPUT_FOCUS {
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

				// forward mouse wheel event to emulation when in playmode
				//
				// * earlier versions of this code also forwarded when
				// hovered over the TV Screen window in the debugging
				// mode. the newer timeline window provides a better
				// interface for rewind
				if img.isPlaymode() && !img.wm.hoverAnyWindowPlaymode() {
					select {
					case input <- userinput.EventMouseWheel{
						Delta: deltaY,
						Mod:   getKeyMod(),
					}:
					default:
						logger.Log(logger.Allow, "sdlimgui", "dropped mouse wheel event")
					}
				} else {
					imgui.CurrentIO().AddMouseWheelDelta(-deltaX/4, deltaY/4)
				}
			}

		case *sdl.JoyButtonEvent:
			img.smartCursorVisibility(true)

			button := userinput.GamepadButtonNone
			switch ev.Button {
			case 0:
				button = userinput.GamepadButtonA
			case 1:
				button = userinput.GamepadButtonB
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
				case input <- userinput.EventGamepadButton{
					ID:     plugging.PortLeft,
					Button: button,
					Down:   ev.State == 1,
				}:
				default:
					logger.Log(logger.Allow, "sdlimgui", "dropped gamepad button event")
				}
			}

		case *sdl.JoyHatEvent:
			img.smartCursorVisibility(true)

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
				case input <- userinput.EventGamepadDPad{
					ID:        plugging.PortLeft,
					Direction: dir,
				}:
				default:
					logger.Log(logger.Allow, "sdlimgui", "dropped gamepad dpad event")
				}
			}

		case *sdl.JoyAxisEvent:
			if img.plt.joysticks[ev.Which].isStelladaptor {
				joy := sdl.JoystickFromInstanceID(ev.Which)
				select {
				case input <- userinput.EventStelladaptor{
					ID:    plugging.PortLeft,
					Horiz: joy.Axis(0),
					Vert:  joy.Axis(1),
				}:
				default:
					logger.Log(logger.Allow, "sdlimgui", "dropped stelladaptor event")
				}
			} else {
				pad := sdl.GameControllerFromInstanceID(ev.Which)
				if pad.Axis(0) > userinput.ThumbstickDeadzone || pad.Axis(0) < -userinput.ThumbstickDeadzone ||
					pad.Axis(1) > userinput.ThumbstickDeadzone || pad.Axis(1) < -userinput.ThumbstickDeadzone ||
					pad.Axis(3) > userinput.ThumbstickDeadzone || pad.Axis(3) < -userinput.ThumbstickDeadzone ||
					pad.Axis(4) > userinput.ThumbstickDeadzone || pad.Axis(4) < -userinput.ThumbstickDeadzone {
					img.smartCursorVisibility(true)
				}

				switch ev.Axis {
				case 0:
					fallthrough
				case 1:
					select {
					case input <- userinput.EventGamepadThumbstick{
						ID:         plugging.PortLeft,
						Thumbstick: userinput.GamepadThumbstickLeft,
						Horiz:      pad.Axis(0),
						Vert:       pad.Axis(1),
					}:
					default:
						logger.Log(logger.Allow, "sdlimgui", "dropped gamepad axis event")
					}
				case 3:
					fallthrough
				case 4:
					select {
					case input <- userinput.EventGamepadThumbstick{
						ID:         plugging.PortLeft,
						Thumbstick: userinput.GamepadThumbstickRight,
						Horiz:      pad.Axis(3),
						Vert:       pad.Axis(4),
					}:
					default:
						logger.Log(logger.Allow, "sdlimgui", "dropped gamepad axis event")
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
					case input <- userinput.EventGamepadTrigger{
						ID:      plugging.PortLeft,
						Trigger: trigger,
						Amount:  ev.Value,
					}:
					default:
						logger.Log(logger.Allow, "sdlimgui", "dropped gamepad axis event")
					}
				}
			}
		}
	} // end of polling loop

	// once peeping has finished we can handle any mouse motion events
	if mouseMotion {
		select {
		case input <- userinput.EventMouseMotion{
			X: int16(mouseMotionX), Y: int16(mouseMotionY),
		}:
		default:
			logger.Log(logger.Allow, "sdlimgui", "dropped mouse motion event")
		}
	}
}

func (img *SdlImgui) renderFrame() {
	img.dbg.PushFunction(func() {
		img.cache.Update(img.dbg.VCS(), img.dbg.Rewind, img.dbg)
	})
	if !img.cache.Resolve() {
		return
	}

	// start of a new frame
	img.plt.newFrame()
	imgui.NewFrame()

	// update metrics
	img.metrics.update()

	// draw all windows according to debug/playmode
	img.draw()

	// rendering
	imgui.Render() // This call only creates the draw data list. Actual rendering to framebuffer is done below.
	img.rnd.preRender()
	img.screen.render()
	img.rnd.render()
	img.plt.postRender()

	// process any functions that should only be done after rendering
	done := false
	for !done {
		select {
		case f := <-img.postRenderFunctions:
			f()
		default:
			done = true
		}
	}
}

// serviceWindowEvent implements the sdl.EventFilter interface
//
// we handle sdl.WindowEvent events in the event filter because there is a bug
// in MacOS/OpenGL which means windows are not refreshed during a window
// resize, only when the resize is finished. this results in poor visual
// feedback
//
// bug described here with suggested fix:
//
// https://stackoverflow.com/questions/34967628/sdl2-window-turns-black-on-resize
func (img *SdlImgui) serviceWindowEvent(ev sdl.Event, userdata any) bool {
	switch ev := ev.(type) {
	case *sdl.WindowEvent:
		switch ev.Event {
		case sdl.WINDOWEVENT_FOCUS_GAINED:
			// the time when the window gained focus see windowFocusedTime
			// declaration for an explanation
			img.windowFocusedTime = time.Now()

		case sdl.WINDOWEVENT_SIZE_CHANGED:
			img.screen.crit.section.Lock()
			img.fonts.resize(img.plt)
			img.playScr.resize()
			img.screen.crit.section.Unlock()
		}

		img.polling.alert = true
	}

	// always return true so that the main service loop can make further
	// decisions based on the WindowEvent
	return true
}
