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

package execution

import (
	"github.com/jetsetilly/gopher2600/curated"
)

// IsValid checks whether the instance of Result contains information
// consistent with the instruction definition.
func (r Result) IsValid() error {
	if !r.Final {
		return curated.Errorf("cpu: execution not finalised (bad opcode?)")
	}

	// is PageFault valid given content of Defn
	if !r.Defn.PageSensitive && r.PageFault {
		return curated.Errorf("cpu: unexpected page fault")
	}

	// byte count
	if r.ByteCount != r.Defn.Bytes {
		return curated.Errorf("cpu: unexpected number of bytes read during decode (%d instead of %d)", r.ByteCount, r.Defn.Bytes)
	}

	// if a bug has been triggered, don't perform the number of cycles check
	if r.CPUBug == "" {
		if r.Defn.IsBranch() {
			if r.Cycles != r.Defn.Cycles && r.Cycles != r.Defn.Cycles+1 && r.Cycles != r.Defn.Cycles+2 {
				return curated.Errorf("cpu: number of cycles wrong for opcode %#02x [%s] (%d instead of %d, %d or %d)",
					r.Defn.OpCode,
					r.Defn.Mnemonic,
					r.Cycles,
					r.Defn.Cycles,
					r.Defn.Cycles+1,
					r.Defn.Cycles+2)
			}
		} else {
			if r.Defn.PageSensitive {
				if r.PageFault && r.Cycles != r.Defn.Cycles && r.Cycles != r.Defn.Cycles+1 {
					return curated.Errorf("cpu: number of cycles wrong for opcode %#02x [%s] (%d instead of %d, %d)",
						r.Defn.OpCode,
						r.Defn.Mnemonic,
						r.Cycles,
						r.Defn.Cycles,
						r.Defn.Cycles+1)
				}
			} else {
				if r.Cycles != r.Defn.Cycles {
					return curated.Errorf("cpu: number of cycles wrong for opcode %#02x [%s] (%d instead of %d)",
						r.Defn.OpCode,
						r.Defn.Mnemonic,
						r.Cycles,
						r.Defn.Cycles)
				}
			}
		}
	}

	return nil
}
