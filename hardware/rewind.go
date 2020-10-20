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

package hardware

import (
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory"
	"github.com/jetsetilly/gopher2600/hardware/riot"
	"github.com/jetsetilly/gopher2600/hardware/tia"
)

type quantum struct {
	CPU  *cpu.CPU
	Mem  *memory.Memory
	TIA  *tia.TIA
	RIOT *riot.RIOT
}

type rewind struct {
	vcs      *VCS
	steps    []quantum
	position int
}

func newRewind(vcs *VCS) rewind {
	return rewind{
		vcs:   vcs,
		steps: make([]quantum, 10000),
	}
}

func (r *rewind) Append() {
	q := quantum{
		CPU:  r.vcs.CPU.Copy(),
		Mem:  r.vcs.Mem.Copy(),
		TIA:  r.vcs.TIA.Copy(),
		RIOT: r.vcs.RIOT.Copy(),
	}

	if r.position >= 10000 {
		r.steps = append(r.steps[1:], q)
	} else {
		r.steps = append(r.steps[:r.position+1], q)
	}
	r.position = len(r.steps) - 1
}

func (r rewind) State() (int, int) {
	return len(r.steps), r.position
}

func (r *rewind) SetPosition(pos int) {
	if pos >= len(r.steps) {
		pos = len(r.steps) - 1
	}
	r.position = pos
	q := r.steps[r.position]
	r.vcs.CPU = q.CPU
	r.vcs.Mem = q.Mem
	r.vcs.TIA = q.TIA
	r.vcs.RIOT = q.RIOT
}
