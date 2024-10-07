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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/veandco/go-sdl2/sdl"
)

// Service implements GuiCreator interface.
func (img *SdlImgui) Service() {
	var err error

	// handle font reset procedure
	if img.resetFonts >= 1 {
		if img.resetFonts == 1 {
			err = img.fonts.initialise(img.plt, img.rnd, img.prefs)
			if err != nil {
				panic(err)
			}
			img.wm.hasInitialised = false
		}
		img.resetFonts--
	}

	// phantom input system must be reset before anything else is processed
	img.phantomInput = phantomInputNone

	// poll for sdl event or timeout
	img.polling.pumpedEvents[0] = img.polling.wait()

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

	// some events we want to service even if an event channel has not been
	// set. very few "modes" will not set a channel but we should be mindful of
	// those events that we handle entirely within this GUI implementation
	//
	// for those events that do require an event channel to be defined we wrap
	// them in select block and log the dropped event in the default case
	if img.polling.pumpedEvents[0] != nil {

		// user input from debugger
		input := img.dbg.UserInput()

		// events are pumped once per frame
		sdl.PumpEvents()

		// the number of additionalEvents is always one on the first iteration
		// of the peeping loop
		additionalEvents := 1

		peeping := true
		for peeping {
			peepCt, err := sdl.PeepEvents(img.polling.pumpedEvents[additionalEvents:], sdl.GETEVENT, sdl.FIRSTEVENT, sdl.LASTEVENT)
			if err != nil {
				logger.Log(logger.Allow, "sdlimgui", err)
			}

			// adjust the peepCt by the number of additionalEvents in the queue
			peepCt += additionalEvents

			// pump more events after procesing if the pumpedEvents queue is full
			peeping = peepCt == len(img.polling.pumpedEvents)

			for i := 0; i < peepCt; i++ {
				switch ev := img.polling.pumpedEvents[i].(type) {
				case *sdl.QuitEvent:
					img.quit()

				// case *sdl.WindowEvent handled by event filter (see comment in serviceWindowEvent()

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

					// trigger service wake in time for next Service() iteration.
					// without this, the results of the mouse button will not be
					// seen until the timeout (in the next iteration) has elapsed.
					//
					// eg. closing a window: the window will be drawn on *this*
					// frame and *this* mouse button press will be acknowledged.
					// next frame the window will not be drawn. however, the *next*
					// frame will sleep until the time out - *this* mouse button
					// event has been consumed. setting alerted ensures there is no
					// delay in drawing the *next* frame
					img.polling.alerted = true

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
			}
		} // end of for peeping

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

	img.renderFrame()
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
func (img *SdlImgui) serviceWindowEvent(ev sdl.Event, userdata interface{}) bool {
	switch ev := ev.(type) {
	case *sdl.WindowEvent:
		switch ev.Event {
		case sdl.WINDOWEVENT_FOCUS_GAINED:
			// the time when the window gained focus see windowFocusedTime
			// declaration for an explanation
			img.windowFocusedTime = time.Now()

		case sdl.WINDOWEVENT_SIZE_CHANGED:
			if img.polling.throttleResize() {
				img.screen.crit.section.Lock()
				img.playScr.setScaling()
				img.screen.crit.section.Unlock()
				img.renderFrame()
			}
		}

		return false
	}
	return true
}
