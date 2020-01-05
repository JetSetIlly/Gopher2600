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
	"gopher2600/debugger/terminal/commandline"
	"gopher2600/errors"
	"gopher2600/television"
	"strings"
)

// defines which types are valid targets
type target interface {
	Label() string

	// the current value of the target. should return a value of type int or
	// bool.
	TargetValue() interface{}

	// format an arbitrary value using suitable formatting method for the target
	FormatValue(val interface{}) string
}

// genericTarget is a way of targetting values that otherwise do not satisfy
// the target interface.
type genericTarget struct {
	label        string
	currentValue interface{}
}

func (trg genericTarget) Label() string {
	return trg.label
}

func (trg genericTarget) TargetValue() interface{} {
	switch v := trg.currentValue.(type) {
	case func() interface{}:
		return v()
	default:
		return v
	}
}

func (trg genericTarget) FormatValue(val interface{}) string {
	switch t := trg.currentValue.(type) {
	case string:
		return val.(string)
	case func() interface{}:
		switch t().(type) {
		case string:
			return val.(string)
		default:
			return fmt.Sprintf("%#v", val)
		}
	default:
		return fmt.Sprintf("%#v", val)
	}
}

// parseTarget interprets the next token and returns a target if it is
// recognised. returns error if it is not.
func parseTarget(dbg *Debugger, tokens *commandline.Tokens) (target, error) {
	var trg target

	keyword, present := tokens.Get()
	if present {
		keyword = strings.ToUpper(keyword)
		switch keyword {
		// cpu registers
		case "PC":
			trg = dbg.vcs.CPU.PC
		case "A":
			trg = dbg.vcs.CPU.A
		case "X":
			trg = dbg.vcs.CPU.X
		case "Y":
			trg = dbg.vcs.CPU.Y
		case "SP":
			trg = dbg.vcs.CPU.SP

		// tv state
		case "FRAMENUM", "FRAME", "FR":
			trg = &genericTarget{
				label: "Frame",
				currentValue: func() interface{} {
					fr, err := dbg.vcs.TV.GetState(television.ReqFramenum)
					if err != nil {
						return err
					}
					return fr
				},
			}
		case "SCANLINE", "SL":
			trg = &genericTarget{
				label: "Scanline",
				currentValue: func() interface{} {
					sl, err := dbg.vcs.TV.GetState(television.ReqScanline)
					if err != nil {
						return err
					}
					return sl
				},
			}
		case "HORIZPOS", "HP":
			trg = &genericTarget{
				label: "Horiz Pos",
				currentValue: func() interface{} {
					hp, err := dbg.vcs.TV.GetState(television.ReqHorizPos)
					if err != nil {
						return err
					}
					return hp
				},
			}

		// cpu instruction targetting was originally added as an experiment, to
		// help investigate a bug in the emulation. I don't think it's much use
		// but it was an instructive exercise and may come in useful one day.
		case "INSTRUCTION", "INS":
			subkey, present := tokens.Get()
			if present {
				subkey = strings.ToUpper(subkey)
				switch subkey {
				case "EFFECT", "EFF":
					trg = &genericTarget{
						label: "Instruction Effect",
						currentValue: func() interface{} {
							if !dbg.vcs.CPU.LastResult.Final {
								return -1
							}
							return int(dbg.vcs.CPU.LastResult.Defn.Effect)
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
