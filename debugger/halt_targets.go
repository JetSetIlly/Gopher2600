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

package debugger

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
)

// targetValue represents the underlying value of the target. for example in
// the case of the CPU program counter target, the underlying type is a uint16.
//
// !!TODO: candidate for generic / comparable type
type targetValue interface{}

type target struct {
	// a label for the label
	label string

	// the current value of the target. note this may not be the same value as the
	// underlying target. for example, target PC will return zero if there is a
	// coprocessor running.
	value func() targetValue

	// the value returned by the value field/function can be formatted for
	// presentation purposes with formatValue()
	format string

	// some targets should only be checked on an instruction boundary
	instructionBoundary bool

	// this target will always break in playmode almost immediately. we use
	// this flag to decide whether to allow the debugger to switch to playmode
	notInPlaymode bool
}

// returns value() formated by the format string. accepts a target value as an
// argument so the format string can be used on any valid targetValue.
func (trg target) stringValue(v targetValue) string {
	if trg.format == "" {
		return fmt.Sprintf("%v", v)
	}
	return fmt.Sprintf(trg.format, v)
}

// parseTarget interprets the next token and returns a target if it is
// recognised. returns error if it is not.
func parseTarget(dbg *Debugger, tokens *commandline.Tokens) (*target, error) {
	var trg *target

	keyword, present := tokens.Get()
	if present {
		keyword = strings.ToUpper(keyword)
		switch keyword {
		// cpu registers
		case "PC":
			trg = &target{
				label: "PC",
				value: func() targetValue {
					// check that there is no coprocessor running - the ARM
					// class of coprocessors cause the 6507 to run NOP
					// instructions, which causes the PC to advance. this means
					// that a PC break can be triggered when the user probably
					// doesn't want it to be.
					bank := dbg.vcs.Mem.Cart.GetBank(dbg.vcs.CPU.PC.Address())
					if bank.ExecutingCoprocessor {
						// we return zero. it's highly unlikely a genuine BREAK PC 0
						// has been requested but even so, this isn't great design.
						// better to return two values perhaps, the second value saying
						// that the target is out of action.
						return 0
					}

					// for breakpoints it is important that the breakpoint
					// value be normalised through MapAddress() too
					ai := dbg.dbgmem.MapAddress(dbg.vcs.CPU.PC.Address(), true)
					return int(ai.MappedAddress)
				},
				format:              "%#04x",
				instructionBoundary: true,
			}

		case "A":
			trg = &target{
				label: "A",
				value: func() targetValue {
					return int(dbg.vcs.CPU.A.Value())
				},
				format:              "%#02x",
				instructionBoundary: true,
			}

		case "X":
			trg = &target{
				label: "X",
				value: func() targetValue {
					return int(dbg.vcs.CPU.X.Value())
				},
				format:              "%#02x",
				instructionBoundary: true,
			}

		case "Y":
			trg = &target{
				label: "Y",
				value: func() targetValue {
					return int(dbg.vcs.CPU.Y.Value())
				},
				format:              "%#02x",
				instructionBoundary: true,
			}

		case "SP":
			trg = &target{
				label: "SP",
				value: func() targetValue {
					return int(dbg.vcs.CPU.SP.Value())
				},
				format:              "%#02x",
				instructionBoundary: true,
			}

		case "RDY":
			trg = &target{
				label: "RDY",
				value: func() targetValue {
					return bool(dbg.vcs.CPU.RdyFlg)
				},
				format:              "%v",
				instructionBoundary: true,
			}

		case "PCZERO":
			trg = &target{
				label: "PCZERO",
				value: func() targetValue {
					return bool(dbg.vcs.CPU.PC.Address() == 0)
				},
				format:              "%v",
				instructionBoundary: true,
			}

		// tv state
		case "FRAMENUM", "FRAME", "FR":
			trg = &target{
				label: "Frame",
				value: func() targetValue {
					return dbg.vcs.TV.GetCoords().Frame
				},
			}

		case "SCANLINE", "SL":
			trg = &target{
				label: "Scanline",
				value: func() targetValue {
					return dbg.vcs.TV.GetCoords().Scanline
				},

				// specifying scanline to be notInPlaymode was considered but
				// this will occasionaly not be true. for example, a scanline
				// value that is beyond the extent of the generated TV frame
			}

		case "CLOCK", "CL":
			trg = &target{
				label: "Clock",
				value: func() targetValue {
					return dbg.vcs.TV.GetCoords().Clock
				},

				// it is impossible to measure the clock value accurately in
				// playmode because the state of the machine is only checked
				// after every CPU instruction
				//
				// another reason not to allow playmode when this halt target
				// is being used, is that the emulation will almost immediately
				// halt - the clock value *will* be reached within 228 ticks of
				// the emulation
				notInPlaymode: true,
			}

		case "BANK":
			trg = bankTarget(dbg)

		// cpu instruction targeting was originally added as an experiment, to
		// help investigate a bug in the emulation. I don't think it's much use
		// but it was an instructive exercise and may come in useful one day.
		case "RESULT", "RES":
			subkey, present := tokens.Get()
			if present {
				subkey = strings.ToUpper(subkey)
				switch subkey {
				case "OPERATOR", "OP":
					trg = &target{
						label: "Operator",
						value: func() targetValue {
							if !dbg.vcs.CPU.LastResult.Final || dbg.vcs.CPU.LastResult.Defn == nil {
								return ""
							}
							return dbg.vcs.CPU.LastResult.Defn.Operator
						},
						instructionBoundary: true,
					}

				case "ADDRESSMODE", "AM":
					trg = &target{
						label: "AddressMode",
						value: func() targetValue {
							if !dbg.vcs.CPU.LastResult.Final || dbg.vcs.CPU.LastResult.Defn == nil {
								return ""
							}
							return int(dbg.vcs.CPU.LastResult.Defn.AddressingMode)
						},
						instructionBoundary: true,
					}

				case "EFFECT", "EFF":
					trg = &target{
						label: "Instruction Effect",
						value: func() targetValue {
							if !dbg.vcs.CPU.LastResult.Final {
								return -1
							}
							return int(dbg.vcs.CPU.LastResult.Defn.Effect)
						},
						instructionBoundary: true,
					}

				case "PAGEFAULT", "PAGE":
					trg = &target{
						label: "PageFault",
						value: func() targetValue {
							return dbg.vcs.CPU.LastResult.PageFault
						},
						instructionBoundary: true,
					}

				case "BUG":
					trg = &target{
						label: "CPU Bug",
						value: func() targetValue {
							s := dbg.vcs.CPU.LastResult.CPUBug
							if s == "" {
								return "ok"
							}
							return s
						},
						instructionBoundary: true,
					}

				default:
					return nil, curated.Errorf("invalid target: %s %s", keyword, subkey)
				}
			} else {
				return nil, curated.Errorf("invalid target: %s", keyword)
			}

		default:
			return nil, curated.Errorf("invalid target: %s", keyword)
		}
	}

	return trg, nil
}

// a bank target is generated automatically by the breakpoints system and also
// explicitly by parseTarget()
func bankTarget(dbg *Debugger) *target {
	return &target{
		label: "Bank",
		value: func() targetValue {
			return dbg.vcs.Mem.Cart.GetBank(dbg.vcs.CPU.PC.Address()).Number
		},
		instructionBoundary: true,
	}
}
