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
	"github.com/jetsetilly/gopher2600/errors"
)

// IsValid checks whether the instance of Result contains information
// consistent with the instruction definition.
func (result Result) IsValid() error {
	if !result.Final {
		return errors.Errorf("cpu: execution not finalised (bad opcode?)")
	}

	// is PageFault valid given content of Defn
	if !result.Defn.PageSensitive && result.PageFault {
		return errors.Errorf("cpu: unexpected page fault")
	}

	// byte count
	if result.ByteCount != result.Defn.Bytes {
		return errors.Errorf("cpu: unexpected number of bytes read during decode (%d instead of %d)", result.ByteCount, result.Defn.Bytes)
	}

	// if a bug has been triggered, don't perform the number of cycles check
	if result.CPUBug == "" {
		if result.Defn.IsBranch() {
			if result.ActualCycles != result.Defn.Cycles && result.ActualCycles != result.Defn.Cycles+1 && result.ActualCycles != result.Defn.Cycles+2 {
				return errors.Errorf("cpu: number of cycles wrong for opcode %#02x [%s] (%d instead of %d, %d or %d)",
					result.Defn.OpCode,
					result.Defn.Mnemonic,
					result.ActualCycles,
					result.Defn.Cycles,
					result.Defn.Cycles+1,
					result.Defn.Cycles+2)
			}
		} else {
			if result.Defn.PageSensitive {
				if result.PageFault && result.ActualCycles != result.Defn.Cycles && result.ActualCycles != result.Defn.Cycles+1 {
					return errors.Errorf("cpu: number of cycles wrong for opcode %#02x [%s] (%d instead of %d, %d)",
						result.Defn.OpCode,
						result.Defn.Mnemonic,
						result.ActualCycles,
						result.Defn.Cycles,
						result.Defn.Cycles+1)
				}
			} else {
				if result.ActualCycles != result.Defn.Cycles {
					return errors.Errorf("cpu: number of cycles wrong for opcode %#02x [%s] (%d instead of %d)",
						result.Defn.OpCode,
						result.Defn.Mnemonic,
						result.ActualCycles,
						result.Defn.Cycles)
				}
			}
		}
	}

	return nil
}
