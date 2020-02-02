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

package reflection

// ReflectPixel contains additional debugging information from the last video cycle.
// it is up to the Renderer to match this up with the last television signal
type ReflectPixel struct {
	Label string

	// Renderer implementations are free to use the color information
	// as they wish (adding alpha information seems a probable scenario).
	Red, Green, Blue, Alpha byte

	// whether the attribute is one that is "instant" or resolves after a
	// short scheduled delay
	Scheduled bool
}

// Renderer implementations accepts ReflectPixel values and associates it in
// some way with the moste recent television signal
type Renderer interface {
	SetReflectPixel(ReflectPixel) error
}

// StubRenderer is a minimal implementation of reflection.StubRenderer
type StubRenderer struct{}

// SetReflectPixel implements the relfection.Renderer interface
func (mpx *StubRenderer) SetReflectPixel(_ ReflectPixel) error {
	return nil
}
