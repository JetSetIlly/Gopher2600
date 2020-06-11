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

package debugger

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/television"
)

type target struct {
	label string

	// must be a comparable type
	currentValue interface{}
	format       string
}

func (trg target) Label() string {
	return trg.label
}

func (trg target) TargetValue() interface{} {
	switch v := trg.currentValue.(type) {
	case func() interface{}:
		return v()
	default:
		return v
	}
}

func (trg target) FormatValue(val interface{}) string {
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
				currentValue: func() interface{} {
					// for breakpoints it is important that the breakpoint
					// value be normalised through mapAddress() too
					ai := dbg.dbgmem.mapAddress(dbg.VCS.CPU.PC.Address(), true)
					return int(ai.mappedAddress)
				},
				format: "%#04x",
			}

		case "A":
			trg = &target{
				label: "A",
				currentValue: func() interface{} {
					return int(dbg.VCS.CPU.A.Value())
				},
				format: "%#02x",
			}

		case "X":
			trg = &target{
				label: "X",
				currentValue: func() interface{} {
					return int(dbg.VCS.CPU.X.Value())
				},
				format: "%#02x",
			}

		case "Y":
			trg = &target{
				label: "Y",
				currentValue: func() interface{} {
					return int(dbg.VCS.CPU.Y.Value())
				},
				format: "%#02x",
			}

		case "SP":
			trg = &target{
				label: "X",
				currentValue: func() interface{} {
					return int(dbg.VCS.CPU.Y.Value())
				},
				format: "%#02x",
			}

		// tv state
		case "FRAMENUM", "FRAME", "FR":
			trg = &target{
				label: "Frame",
				currentValue: func() interface{} {
					fr, err := dbg.VCS.TV.GetState(television.ReqFramenum)
					if err != nil {
						return err
					}
					return fr
				},
			}

		case "SCANLINE", "SL":
			trg = &target{
				label: "Scanline",
				currentValue: func() interface{} {
					sl, err := dbg.VCS.TV.GetState(television.ReqScanline)
					if err != nil {
						return err
					}
					return sl
				},
			}

		case "HORIZPOS", "HP":
			trg = &target{
				label: "Horiz Pos",
				currentValue: func() interface{} {
					hp, err := dbg.VCS.TV.GetState(television.ReqHorizPos)
					if err != nil {
						return err
					}
					return hp
				},
			}

		case "BANK":
			trg = bankTarget(dbg)

		// cpu instruction targetting was originally added as an experiment, to
		// help investigate a bug in the emulation. I don't think it's much use
		// but it was an instructive exercise and may come in useful one day.
		case "RESULT", "RES":
			subkey, present := tokens.Get()
			if present {
				subkey = strings.ToUpper(subkey)
				switch subkey {
				case "MNEMONIC", "MNE":
					trg = &target{
						label: "Mnemonic",
						currentValue: func() interface{} {
							if !dbg.VCS.CPU.LastResult.Final || dbg.VCS.CPU.LastResult.Defn == nil {
								return ""
							}
							return dbg.VCS.CPU.LastResult.Defn.Mnemonic
						},
					}

				case "ADDRESSMODE", "AM":
					trg = &target{
						label: "AddressMode",
						currentValue: func() interface{} {
							if !dbg.VCS.CPU.LastResult.Final || dbg.VCS.CPU.LastResult.Defn == nil {
								return ""
							}
							return int(dbg.VCS.CPU.LastResult.Defn.AddressingMode)
						},
					}

				case "EFFECT", "EFF":
					trg = &target{
						label: "Instruction Effect",
						currentValue: func() interface{} {
							if !dbg.VCS.CPU.LastResult.Final {
								return -1
							}
							return int(dbg.VCS.CPU.LastResult.Defn.Effect)
						},
					}

				case "PAGEFAULT", "PAGE":
					trg = &target{
						label: "PageFault",
						currentValue: func() interface{} {
							return dbg.VCS.CPU.LastResult.PageFault
						},
					}

				case "BUG":
					trg = &target{
						label: "CPU Bug",
						currentValue: func() interface{} {
							s := dbg.VCS.CPU.LastResult.CPUBug
							if s == "" {
								return "ok"
							}
							return s
						},
					}

				default:
					return nil, errors.New(errors.InvalidTarget, fmt.Sprintf("%s %s", keyword, subkey))
				}
			} else {
				return nil, errors.New(errors.InvalidTarget, keyword)
			}

		default:
			return nil, errors.New(errors.InvalidTarget, keyword)
		}
	}

	return trg, nil
}

func bankTarget(dbg *Debugger) *target {
	return &target{
		label: "Bank",
		currentValue: func() interface{} {
			return int(dbg.VCS.Mem.Cart.GetBank(dbg.VCS.CPU.PC.Address()))
		},
	}
}
