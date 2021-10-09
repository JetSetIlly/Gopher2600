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
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
)

// targetValue represents the value that is to be monitored.
type targetValue interface{}

type target struct {
	label string

	// must be a comparable type
	currentValue targetValue
	format       string

	// some targets should only be checked on an instruction boundary
	instructionBoundary bool
}

func (trg target) Label() string {
	return trg.label
}

func (trg target) TargetValue() targetValue {
	switch v := trg.currentValue.(type) {
	case func() targetValue:
		return v()
	default:
		return v
	}
}

func (trg target) FormatValue(val targetValue) string {
	if trg.format == "" {
		return fmt.Sprintf("%v", val)
	}
	return fmt.Sprintf(trg.format, val)
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
				currentValue: func() targetValue {
					// check that there is no coprocessor runnig - the ARM
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
					// value be normalised through mapAddress() too
					ai := dbg.dbgmem.mapAddress(dbg.vcs.CPU.PC.Address(), true)
					return int(ai.mappedAddress)
				},
				format:              "%#04x",
				instructionBoundary: true,
			}

		case "A":
			trg = &target{
				label: "A",
				currentValue: func() targetValue {
					return int(dbg.vcs.CPU.A.Value())
				},
				format:              "%#02x",
				instructionBoundary: false,
			}

		case "X":
			trg = &target{
				label: "X",
				currentValue: func() targetValue {
					return int(dbg.vcs.CPU.X.Value())
				},
				format:              "%#02x",
				instructionBoundary: false,
			}

		case "Y":
			trg = &target{
				label: "Y",
				currentValue: func() targetValue {
					return int(dbg.vcs.CPU.Y.Value())
				},
				format:              "%#02x",
				instructionBoundary: false,
			}

		case "SP":
			trg = &target{
				label: "SP",
				currentValue: func() targetValue {
					return int(dbg.vcs.CPU.SP.Value())
				},
				format:              "%#02x",
				instructionBoundary: false,
			}

		// tv state
		case "FRAMENUM", "FRAME", "FR":
			trg = &target{
				label: "Frame",
				currentValue: func() targetValue {
					return dbg.vcs.TV.GetState(signal.ReqFramenum)
				},
				instructionBoundary: false,
			}

		case "SCANLINE", "SL":
			trg = &target{
				label: "Scanline",
				currentValue: func() targetValue {
					return dbg.vcs.TV.GetState(signal.ReqScanline)
				},
				instructionBoundary: false,
			}

		case "CLOCK", "CL":
			trg = &target{
				label: "Clock",
				currentValue: func() targetValue {
					return dbg.vcs.TV.GetState(signal.ReqClock)
				},
				instructionBoundary: false,
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
						currentValue: func() targetValue {
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
						currentValue: func() targetValue {
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
						currentValue: func() targetValue {
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
						currentValue: func() targetValue {
							return dbg.vcs.CPU.LastResult.PageFault
						},
						instructionBoundary: true,
					}

				case "BUG":
					trg = &target{
						label: "CPU Bug",
						currentValue: func() targetValue {
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
// explicitly in parseTarget().
func bankTarget(dbg *Debugger) *target {
	return &target{
		label: "Bank",
		currentValue: func() targetValue {
			return dbg.vcs.Mem.Cart.GetBank(dbg.vcs.CPU.PC.Address()).Number
		},
		instructionBoundary: false,
	}
}