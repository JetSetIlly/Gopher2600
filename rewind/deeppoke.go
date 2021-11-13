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

package rewind

import "github.com/jetsetilly/gopher2600/curated"

type PokeHook func(res *State) error

// RunPoke will the run the VCS from one state to another state applying
// the supplied PokeHook to the from State
func (r *Rewind) RunPoke(from *State, to *State, poke PokeHook) error {
	fromIdx := r.findFrameIndex(from.TV.GetCoords().Frame).fromIdx

	if poke != nil {
		err := poke(r.entries[fromIdx])
		if err != nil {
			return err
		}
	}

	err := r.setSplicePoint(fromIdx, to.TV.GetCoords())
	if err != nil {
		return curated.Errorf("rewind: %v", err)
	}

	return nil
}
