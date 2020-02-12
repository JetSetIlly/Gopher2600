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

package execution

import (
	"fmt"
	"gopher2600/errors"
	"reflect"
)

// IsValid checks whether the instance of Result contains information
// consistent with the instruction definition.
func (result Result) IsValid() error {
	if !result.Final {
		return errors.New(errors.InvalidResult, "not checking an unfinalised InstructionResult")
	}

	// check that InstructionData is broadly sensible - is either nil, a uint16 or uint8
	if result.InstructionData != nil {
		ot := reflect.TypeOf(result.InstructionData).Kind()
		if ot != reflect.Uint16 && ot != reflect.Uint8 {
			return errors.New(errors.InvalidResult, fmt.Sprintf("instruction data is bad (%s)", ot))
		}
	}

	// is PageFault valid given content of Defn
	if !result.Defn.PageSensitive && result.PageFault {
		return errors.New(errors.InvalidResult, "unexpected page fault")
	}

	// if a bug has been triggered, don't perform the number of cycles check
	if result.CPUBug == "" {
		if result.Defn.IsBranch() {
			if result.ActualCycles != result.Defn.Cycles && result.ActualCycles != result.Defn.Cycles+1 && result.ActualCycles != result.Defn.Cycles+2 {
				msg := fmt.Sprintf("number of cycles wrong for opcode %#02x [%s] (%d instead of %d, %d or %d)",
					result.Defn.OpCode,
					result.Defn.Mnemonic,
					result.ActualCycles,
					result.Defn.Cycles,
					result.Defn.Cycles+1,
					result.Defn.Cycles+2)
				return errors.New(errors.InvalidResult, msg)
			}
		} else {
			if result.Defn.PageSensitive {
				if result.PageFault && result.ActualCycles != result.Defn.Cycles && result.ActualCycles != result.Defn.Cycles+1 {
					msg := fmt.Sprintf("number of cycles wrong for opcode %#02x [%s] (%d instead of %d, %d)",
						result.Defn.OpCode,
						result.Defn.Mnemonic,
						result.ActualCycles,
						result.Defn.Cycles,
						result.Defn.Cycles+1)
					return errors.New(errors.InvalidResult, msg)
				}
			} else {
				if result.ActualCycles != result.Defn.Cycles {
					msg := fmt.Sprintf("number of cycles wrong for opcode %#02x [%s] (%d instead of %d)",
						result.Defn.OpCode,
						result.Defn.Mnemonic,
						result.ActualCycles,
						result.Defn.Cycles)
					return errors.New(errors.InvalidResult, msg)
				}
			}
		}
	}

	return nil
}
