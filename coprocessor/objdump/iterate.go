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

package objdump

// IterationItemID distinguishes the infromation returned in an IterationItem
// by an ongoing Iteration.
type IterationItemID int

// List of valid IterationItemID values.
const (
	SourceFile IterationItemID = iota
	SourceLine
	AsmLine
	End
)

// iterationItem is sent over the Next channel of an ongoing Iteration.
type IterationItem struct {
	ID      IterationItemID
	Content string
	Detail  bool
}

// Iteration represents an ongoing iteration of the objdump structure.
//
// Iteration doesn't work well with imgui.ListClipper and is
// probably goroutine unsafe - although there are no race conditions
// because objdump doesn't change after intialisation - it might race if
// a new ROMs is inserted and an iteration is taking place.
type Iteration struct {
	Next chan IterationItem
	End  chan bool
}

// NewIteration begins a new interation starting a goroutine to service the
// channels.
func (obj *ObjDump) NewIteration() Iteration {
	it := Iteration{
		Next: make(chan IterationItem),
		End:  make(chan bool),
	}

	go func() {
		done := false
		for _, fn := range obj.files_key {
			f := obj.files[fn]

			it.Next <- IterationItem{
				ID:      SourceFile,
				Content: fn,
				Detail:  len(f.lines) > 0,
			}

			for _, ln := range f.lines {
				it.Next <- IterationItem{
					ID:      SourceLine,
					Content: ln.content,
					Detail:  len(ln.asm) > 0,
				}

				for _, asm := range ln.asm {
					it.Next <- IterationItem{
						ID:      AsmLine,
						Content: asm.asm,
					}
				}
			}

			select {
			case done = <-it.End:
			default:
			}

			if done {
				break // for loop
			}
		}

		if !done {
			it.Next <- IterationItem{
				ID: End,
			}
		}
	}()

	return it
}
