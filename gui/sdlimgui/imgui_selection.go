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

type imguiSelection struct {
	a int
	b int
}

func (sel *imguiSelection) clear() {
	sel.a = -1
	sel.b = -1
}

func (sel *imguiSelection) dragStart(i int) {
	sel.a = i
	sel.b = i
}
func (sel *imguiSelection) drag(i int) {
	sel.b = i
}

func (sel imguiSelection) isSingle() bool {
	return sel.a == sel.b
}

func (sel imguiSelection) inRange(i int) bool {
	if sel.b < sel.a {
		return i >= sel.b && i <= sel.a
	}
	return i >= sel.a && i <= sel.b
}

func (sel imguiSelection) limits() (int, int) {
	if sel.a < sel.b {
		return sel.a, sel.b
	}
	return sel.b, sel.a
}
