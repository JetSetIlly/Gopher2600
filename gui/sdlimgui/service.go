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
	"strings"
	"time"
	"unicode"

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/notifications"
	"github.com/jetsetilly/gopher2600/userinput"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/veandco/go-sdl2/sdl"
)

// var last time.Time

// Service implements GuiCreator interface.
func (img *SdlImgui) Service() {
	// handle font reset procedure
	if img.resetFonts >= 1 {
		if img.resetFonts == 1 {
			err := img.glsl.setupFonts()
			if err != nil {
				panic(err)
			}
			img.wm.hasInitialised = false
		}
		img.resetFonts--
	}

	// refresh lazy values
	switch img.mode.Load().(govern.Mode) {
	case govern.ModeDebugger:
		img.lz.Refresh()
	case govern.ModePlay:
		img.lz.FastRefresh()
	}

	// poll for sdl event or timeout
	img.polling.pumpedEvents[0] = img.polling.wait()

	// whether mouse button down event have been polled. if it has and we poll
	// an up event in the same PollEvent() loop below, then we need to
	// "trickle" the up and down events over two frames. see commentary for
	// trickleMouseButton type
	var leftMouseDownPolled bool
	var rightMouseDownPolled bool

	// send only one mouse event to the emulation per frame. the mouseMotion
	// variables below help with this
	var mouseMotionProcess bool
	var mouseMotionX int32
	var mouseMotionY int32

	// some events we want to service even if an event channel has not been
	// set. very few "modes" will not set a channel but we should be mindful of
	// those events that we handle entirely within this GUI implementation
	//
	// for those events that do require an event channel to be defined we wrap
	// them in select block and log the dropped event in the default case
	if img.polling.pumpedEvents[0] != nil {

		// events are pumped once per frame
		sdl.PumpEvents()

		// the number of additionalEvents is always one on the first iteration
		// of the peeping loop
		additionalEvents := 1

		peeping := true
		for peeping {
			peepCt, err := sdl.PeepEvents(img.polling.pumpedEvents[additionalEvents:], sdl.GETEVENT, sdl.FIRSTEVENT, sdl.LASTEVENT)
			if err != nil {
				logger.Log("sdlimgui", err.Error())
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
					if img.hasModal || !img.isCaptured() {
						imgui.CurrentIO().AddInputCharacters(string(ev.Text[:]))
					}

				case *sdl.KeyboardEvent:
					img.smartHideCursor(true)
					img.serviceKeyboard(ev)

				case *sdl.MouseMotionEvent:
					img.smartHideCursor(false)
					if img.isCaptured() {
						mouseMotionX, mouseMotionY, _ = sdl.GetMouseState()
						mouseMotionProcess = mouseMotionX != img.mouseX || mouseMotionY != img.mouseY
					}

				case *sdl.MouseButtonEvent:
					img.smartHideCursor(false)

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
								img.setCapture(false)
								if !img.isPlaymode() {
									img.term.pushCommand("HALT")
								}
							} else if img.isPlaymode() {
								// set mouse capture if mouse is not over a window
								if !img.wm.playmodeHover {
									img.setCapture(true)
								}
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
					// event has been consumed. setting alerted ensures there is no
					// delay in drawing the *next* frame
					img.polling.alerted = true

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
					imgui.CurrentIO().AddMouseWheelDelta(-deltaX/4, deltaY/4)

					if img.mode.Load().(govern.Mode) != govern.ModePlay || !img.wm.playmodeWindows[winSelectROMID].playmodeIsOpen() {
						select {
						case img.userinput <- userinput.EventMouseWheel{Delta: deltaY}:
						default:
							logger.Log("sdlimgui", "dropped mouse wheel event")
						}
					}

				case *sdl.JoyButtonEvent:
					img.smartHideCursor(true)

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
						case img.userinput <- userinput.EventGamepadButton{
							ID:     plugging.PortLeft,
							Button: button,
							Down:   ev.State == 1,
						}:
						default:
							logger.Log("sdlimgui", "dropped gamepad button event")
						}
					}

				case *sdl.JoyHatEvent:
					img.smartHideCursor(true)

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
							ID:        plugging.PortLeft,
							Direction: dir,
						}:
						default:
							logger.Log("sdlimgui", "dropped gamepad dpad event")
						}
					}

				case *sdl.JoyAxisEvent:
					if img.plt.joysticks[ev.Which].isStelladaptor {
						joy := sdl.JoystickFromInstanceID(ev.Which)
						select {
						case img.userinput <- userinput.EventStelladaptor{
							ID:    plugging.PortLeft,
							Horiz: joy.Axis(0),
							Vert:  joy.Axis(1),
						}:
						default:
							logger.Log("sdlimgui", "dropped stelladaptor event")
						}
					} else {
						pad := sdl.GameControllerFromInstanceID(ev.Which)
						if pad.Axis(0) > userinput.ThumbstickDeadzone || pad.Axis(0) < -userinput.ThumbstickDeadzone ||
							pad.Axis(1) > userinput.ThumbstickDeadzone || pad.Axis(1) < -userinput.ThumbstickDeadzone ||
							pad.Axis(3) > userinput.ThumbstickDeadzone || pad.Axis(3) < -userinput.ThumbstickDeadzone ||
							pad.Axis(4) > userinput.ThumbstickDeadzone || pad.Axis(4) < -userinput.ThumbstickDeadzone {
							img.smartHideCursor(true)
						}

						switch ev.Axis {
						case 0:
							fallthrough
						case 1:
							select {
							case img.userinput <- userinput.EventGamepadThumbstick{
								ID:         plugging.PortLeft,
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
								ID:         plugging.PortLeft,
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
								ID:      plugging.PortLeft,
								Trigger: trigger,
								Amount:  ev.Value,
							}:
							default:
								logger.Log("sdlimgui", "dropped gamepad axis event")
							}
						}
					}
				}
			}
		} // end of for peeping

		if mouseMotionProcess {
			w, h := img.plt.window.GetSize()

			// reduce mouse x and y coordintes to the range 0.0 to 1.0
			//  no need to worry about negative numbers and numbers greater
			//  than 1.0 because we (should) have restricted mouse movement
			//  to the window (with window.SetGrab(). see the ReqCaptureMouse
			//  case in the ReqFeature() function)
			x := float32(mouseMotionX) / float32(w)
			y := float32(mouseMotionY) / float32(h)

			// forward new position to emulation
			select {
			case img.userinput <- userinput.EventMouseMotion{
				X: x,
				Y: y,
			}:
			default:
				logger.Log("sdlimgui", "dropped mouse motion event")
			}

			// store mouse position for future frames
			img.mouseX = mouseMotionX
			img.mouseY = mouseMotionY
		}
	}

	img.renderFrame()
}

func (img *SdlImgui) renderFrame() {
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

func (img *SdlImgui) serviceKeyboard(ev *sdl.KeyboardEvent) {
	if ev.Repeat == 1 {
		return
	}

	ctrl := ev.Keysym.Mod&sdl.KMOD_LCTRL == sdl.KMOD_LCTRL || ev.Keysym.Mod&sdl.KMOD_RCTRL == sdl.KMOD_RCTRL
	shift := ev.Keysym.Mod&sdl.KMOD_LSHIFT == sdl.KMOD_LSHIFT || ev.Keysym.Mod&sdl.KMOD_RSHIFT == sdl.KMOD_RSHIFT

	// enable fuzzy window matching based on keyboard modifiers
	img.wm.fuzzySelection = ctrl && shift

	if ev.Type == sdl.KEYUP {

		// special handling if fuzzy windows is enabled
		if img.wm.fuzzySelection {
			switch ev.Keysym.Scancode {
			case sdl.SCANCODE_BACKSPACE:
				if len(img.wm.fuzzyMatch) > 0 {
					img.wm.fuzzyMatch = img.wm.fuzzyMatch[:len(img.wm.fuzzyMatch)-1]
				}
			case sdl.SCANCODE_SPACE:
				// dont allow leading space
				if len(img.wm.fuzzyMatch) > 0 {
					img.wm.fuzzyMatch = fmt.Sprintf("%s ", img.wm.fuzzyMatch)
				}
			default:
				key := sdl.GetKeyFromScancode(ev.Keysym.Scancode)
				if unicode.IsPrint(rune(key)) {
					name := sdl.GetScancodeName(ev.Keysym.Scancode)
					img.wm.fuzzyMatch = fmt.Sprintf("%s%s", img.wm.fuzzyMatch, strings.ToLower(name))
				}
			}
			return
		}

		handled := true

		if img.isPlaymode() {
			switch ev.Keysym.Scancode {
			case sdl.SCANCODE_ESCAPE:
				if img.isCaptured() {
					img.setCapture(false)
				} else if img.wm.playmodeWindows[winSelectROMID].playmodeIsOpen() {
					img.wm.playmodeWindows[winSelectROMID].playmodeSetOpen(false)
				} else {
					img.quit()
				}

			case sdl.SCANCODE_F7:
				img.playScr.toggleFPS()

			default:
				handled = false
			}
		}

		switch ev.Keysym.Scancode {
		case sdl.SCANCODE_TAB:
			if !img.isPlaymode() && imgui.IsAnyItemActive() {
				// in debugger mode do not handle if an imgui widget is active
				// (see the sdl.KEYDOWN branch below for opposite condition and
				// explanation)
				handled = false
			} else {
				if ctrl {
					img.dbg.ReloadCartridge()
				} else {
					// only open ROM selector if window has been focused for a
					// while. see windowFocusedTime declaration for an explanation
					if time.Since(img.windowFocusedTime) > 500*time.Millisecond {
						img.wm.toggleOpen(winSelectROMID)
					}
				}
			}

		case sdl.SCANCODE_GRAVE:
			if img.isPlaymode() {
				img.dbg.PushSetMode(govern.ModeDebugger)
			} else {
				img.dbg.PushSetMode(govern.ModePlay)
			}

		case sdl.SCANCODE_F8:
			w := img.wm.playmodeWindows[winBotID]
			w.playmodeSetOpen(!w.playmodeIsOpen())

		case sdl.SCANCODE_F9:
			img.wm.toggleOpen(winTrackerID)

		case sdl.SCANCODE_F10:
			img.wm.toggleOpen(winPrefsID)

		case sdl.SCANCODE_F11:
			img.prefs.fullScreen.Set(!img.prefs.fullScreen.Get().(bool))

		case sdl.SCANCODE_F12:
			if ctrl && !shift {
				img.glsl.shaders[playscrShaderID].(*playscrShader).screenshot.startProcess(modeComposite, "")
			} else if shift && !ctrl {
				img.glsl.shaders[playscrShaderID].(*playscrShader).screenshot.startProcess(modeMotion, "")
			} else {
				img.glsl.shaders[playscrShaderID].(*playscrShader).screenshot.startProcess(modeSingle, "")
			}

			img.playScr.emulationNotice.set(notifications.NotifyScreenshot)

		case sdl.SCANCODE_F14:
			fallthrough
		case sdl.SCANCODE_SCROLLLOCK:
			img.setCapture(!img.isCaptured())

		case sdl.SCANCODE_F15:
			fallthrough
		case sdl.SCANCODE_PAUSE:
			if img.isPlaymode() {
				if img.dbg.State() == govern.Paused {
					img.dbg.PushSetPause(false)
				} else {
					img.dbg.PushSetPause(true)
				}
			} else {
				if img.dbg.State() == govern.Paused {
					img.term.pushCommand("RUN")
				} else {
					img.term.pushCommand("HALT")
				}
			}

		case sdl.SCANCODE_A:
			if ctrl {
				img.wm.arrangeBySize = 1
			} else {
				handled = false
			}

		case sdl.SCANCODE_R:
			if ctrl {
				img.dbg.ReloadCartridge()
			} else {
				handled = false
			}

		case sdl.SCANCODE_M:
			if ctrl {
				img.toggleAudioMute()
			} else {
				handled = false
			}

		default:
			handled = false
		}

		if handled {
			return
		}
	} else if ev.Type == sdl.KEYDOWN {
		// for debugger mode we test for the ESC key press on the down event
		// and not the up event. this is because imgui widgets react to the ESC
		// key on the down event and we only want to perform our special ESC
		// key handling if no widget is active
		//
		// if we perform out special handling on the up stroke then the active
		// widget will be unselected and then the special handling perfomed on
		// every ESC KEY press. we don't want that. we want the active widget
		// to be deselected and for the special handling to require a
		// completely separate key press

		if !img.isPlaymode() {
			switch ev.Keysym.Scancode {
			case sdl.SCANCODE_TAB:
				// in debugger mode do not handle if an imgui widget is not
				// active (see the sdl.KEYUP branch above for opposite
				// condition)
				//
				// this prevents a KEYDOWN being forwarded to imgui and without
				// the corresponding KEYUP if the TAB key was consumed becaue
				// IsAnyItemActive() was true at time of KEYUP. without this
				// check imgui thinks the TAB key is being held down
				if !imgui.IsAnyItemActive() {
					return
				}
			case sdl.SCANCODE_ESCAPE:
				if !imgui.IsAnyItemActive() {
					if img.isCaptured() {
						img.setCapture(false)
						img.term.pushCommand("HALT")
						return
					} else {
						img.setCapture(true)
						img.term.pushCommand("RUN")
						return
					}
				}
			}
		}
	}

	// forward keypresses to userinput.Event channel
	if img.isCaptured() || (img.isPlaymode() && !imgui.IsAnyItemActive()) {
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
				logger.Log("sdlimgui", "dropped keyboard event")
			}
		}
	}

	// remaining keypresses forwarded to imgui io system
	switch ev.Type {
	case sdl.KEYDOWN:
		imgui.CurrentIO().KeyPress(int(ev.Keysym.Scancode))
		img.updateKeyModifier()
	case sdl.KEYUP:
		imgui.CurrentIO().KeyRelease(int(ev.Keysym.Scancode))
		img.updateKeyModifier()
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
