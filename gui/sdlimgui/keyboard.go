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
	"github.com/jetsetilly/imgui-go/v5"
	"github.com/veandco/go-sdl2/sdl"
)

func sdl2KeyEventToImguiKey(keycode sdl.Keycode, scancode sdl.Scancode) imgui.ImguiKey {
	switch keycode {
	case sdl.K_TAB:
		return imgui.KeyTab
	case sdl.K_LEFT:
		return imgui.KeyLeftArrow
	case sdl.K_RIGHT:
		return imgui.KeyRightArrow
	case sdl.K_UP:
		return imgui.KeyUpArrow
	case sdl.K_DOWN:
		return imgui.KeyDownArrow
	case sdl.K_PAGEUP:
		return imgui.KeyPageUp
	case sdl.K_PAGEDOWN:
		return imgui.KeyPageDown
	case sdl.K_HOME:
		return imgui.KeyHome
	case sdl.K_END:
		return imgui.KeyEnd
	case sdl.K_INSERT:
		return imgui.KeyInsert
	case sdl.K_DELETE:
		return imgui.KeyDelete
	case sdl.K_BACKSPACE:
		return imgui.KeyBackspace
	case sdl.K_SPACE:
		return imgui.KeySpace
	case sdl.K_RETURN:
		return imgui.KeyEnter
	case sdl.K_ESCAPE:
		return imgui.KeyEscape
	//case sdl.K_QUOTE: return imgui.KeyApostrophe
	case sdl.K_COMMA:
		return imgui.KeyComma
	//case sdl.K_MINUS: return imgui.KeyMinus
	case sdl.K_PERIOD:
		return imgui.KeyPeriod
	//case sdl.K_SLASH: return imgui.KeySlash
	case sdl.K_SEMICOLON:
		return imgui.KeySemicolon
	//case sdl.K_EQUALS: return imgui.KeyEqual
	//case sdl.K_LEFTBRACKET: return imgui.KeyLeftBracket
	//case sdl.K_BACKSLASH: return imgui.KeyBackslash
	//case sdl.K_RIGHTBRACKET: return imgui.KeyRightBracket
	//case sdl.K_BACKQUOTE: return imgui.KeyGraveAccent
	case sdl.K_CAPSLOCK:
		return imgui.KeyCapsLock
	case sdl.K_SCROLLLOCK:
		return imgui.KeyScrollLock
	case sdl.K_NUMLOCKCLEAR:
		return imgui.KeyNumLock
	case sdl.K_PRINTSCREEN:
		return imgui.KeyPrintScreen
	case sdl.K_PAUSE:
		return imgui.KeyPause
	case sdl.K_KP_0:
		return imgui.KeyKeypad0
	case sdl.K_KP_1:
		return imgui.KeyKeypad1
	case sdl.K_KP_2:
		return imgui.KeyKeypad2
	case sdl.K_KP_3:
		return imgui.KeyKeypad3
	case sdl.K_KP_4:
		return imgui.KeyKeypad4
	case sdl.K_KP_5:
		return imgui.KeyKeypad5
	case sdl.K_KP_6:
		return imgui.KeyKeypad6
	case sdl.K_KP_7:
		return imgui.KeyKeypad7
	case sdl.K_KP_8:
		return imgui.KeyKeypad8
	case sdl.K_KP_9:
		return imgui.KeyKeypad9
	case sdl.K_KP_PERIOD:
		return imgui.KeyKeypadDecimal
	case sdl.K_KP_DIVIDE:
		return imgui.KeyKeypadDivide
	case sdl.K_KP_MULTIPLY:
		return imgui.KeyKeypadMultiply
	case sdl.K_KP_MINUS:
		return imgui.KeyKeypadSubtract
	case sdl.K_KP_PLUS:
		return imgui.KeyKeypadAdd
	case sdl.K_KP_ENTER:
		return imgui.KeyKeypadEnter
	case sdl.K_KP_EQUALS:
		return imgui.KeyKeypadEqual
	case sdl.K_LCTRL:
		return imgui.KeyLeftCtrl
	case sdl.K_LSHIFT:
		return imgui.KeyLeftShift
	case sdl.K_LALT:
		return imgui.KeyLeftAlt
	case sdl.K_LGUI:
		return imgui.KeyLeftSuper
	case sdl.K_RCTRL:
		return imgui.KeyRightCtrl
	case sdl.K_RSHIFT:
		return imgui.KeyRightShift
	case sdl.K_RALT:
		return imgui.KeyRightAlt
	case sdl.K_RGUI:
		return imgui.KeyRightSuper
	case sdl.K_APPLICATION:
		return imgui.KeyMenu
	case sdl.K_0:
		return imgui.Key0
	case sdl.K_1:
		return imgui.Key1
	case sdl.K_2:
		return imgui.Key2
	case sdl.K_3:
		return imgui.Key3
	case sdl.K_4:
		return imgui.Key4
	case sdl.K_5:
		return imgui.Key5
	case sdl.K_6:
		return imgui.Key6
	case sdl.K_7:
		return imgui.Key7
	case sdl.K_8:
		return imgui.Key8
	case sdl.K_9:
		return imgui.Key9
	case sdl.K_a:
		return imgui.KeyA
	case sdl.K_b:
		return imgui.KeyB
	case sdl.K_c:
		return imgui.KeyC
	case sdl.K_d:
		return imgui.KeyD
	case sdl.K_e:
		return imgui.KeyE
	case sdl.K_f:
		return imgui.KeyF
	case sdl.K_g:
		return imgui.KeyG
	case sdl.K_h:
		return imgui.KeyH
	case sdl.K_i:
		return imgui.KeyI
	case sdl.K_j:
		return imgui.KeyJ
	case sdl.K_k:
		return imgui.KeyK
	case sdl.K_l:
		return imgui.KeyL
	case sdl.K_m:
		return imgui.KeyM
	case sdl.K_n:
		return imgui.KeyN
	case sdl.K_o:
		return imgui.KeyO
	case sdl.K_p:
		return imgui.KeyP
	case sdl.K_q:
		return imgui.KeyQ
	case sdl.K_r:
		return imgui.KeyR
	case sdl.K_s:
		return imgui.KeyS
	case sdl.K_t:
		return imgui.KeyT
	case sdl.K_u:
		return imgui.KeyU
	case sdl.K_v:
		return imgui.KeyV
	case sdl.K_w:
		return imgui.KeyW
	case sdl.K_x:
		return imgui.KeyX
	case sdl.K_y:
		return imgui.KeyY
	case sdl.K_z:
		return imgui.KeyZ
	case sdl.K_F1:
		return imgui.KeyF1
	case sdl.K_F2:
		return imgui.KeyF2
	case sdl.K_F3:
		return imgui.KeyF3
	case sdl.K_F4:
		return imgui.KeyF4
	case sdl.K_F5:
		return imgui.KeyF5
	case sdl.K_F6:
		return imgui.KeyF6
	case sdl.K_F7:
		return imgui.KeyF7
	case sdl.K_F8:
		return imgui.KeyF8
	case sdl.K_F9:
		return imgui.KeyF9
	case sdl.K_F10:
		return imgui.KeyF10
	case sdl.K_F11:
		return imgui.KeyF11
	case sdl.K_F12:
		return imgui.KeyF12
	case sdl.K_F13:
		return imgui.KeyF13
	case sdl.K_F14:
		return imgui.KeyF14
	case sdl.K_F15:
		return imgui.KeyF15
	case sdl.K_F16:
		return imgui.KeyF16
	case sdl.K_F17:
		return imgui.KeyF17
	case sdl.K_F18:
		return imgui.KeyF18
	case sdl.K_F19:
		return imgui.KeyF19
	case sdl.K_F20:
		return imgui.KeyF20
	case sdl.K_F21:
		return imgui.KeyF21
	case sdl.K_F22:
		return imgui.KeyF22
	case sdl.K_F23:
		return imgui.KeyF23
	case sdl.K_F24:
		return imgui.KeyF24
	case sdl.K_AC_BACK:
		return imgui.KeyAppBack
	case sdl.K_AC_FORWARD:
		return imgui.KeyAppForward
	}

	// Fallback to scancode
	switch scancode {
	case sdl.SCANCODE_GRAVE:
		return imgui.KeyGraveAccent
	case sdl.SCANCODE_MINUS:
		return imgui.KeyMinus
	case sdl.SCANCODE_EQUALS:
		return imgui.KeyEqual
	case sdl.SCANCODE_LEFTBRACKET:
		return imgui.KeyLeftBracket
	case sdl.SCANCODE_RIGHTBRACKET:
		return imgui.KeyRightBracket
	case sdl.SCANCODE_NONUSBACKSLASH:
		return imgui.KeyOem102
	case sdl.SCANCODE_BACKSLASH:
		return imgui.KeyBackslash
	case sdl.SCANCODE_SEMICOLON:
		return imgui.KeySemicolon
	case sdl.SCANCODE_APOSTROPHE:
		return imgui.KeyApostrophe
	case sdl.SCANCODE_COMMA:
		return imgui.KeyComma
	case sdl.SCANCODE_PERIOD:
		return imgui.KeyPeriod
	case sdl.SCANCODE_SLASH:
		return imgui.KeySlash
	}

	return imgui.KeyNone
}
