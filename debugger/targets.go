package debugger

import (
	"fmt"
	"gopher2600/debugger/commandline"
	"gopher2600/errors"
	"gopher2600/television"
	"strings"
)

// defines which types are valid targets
type target interface {
	Label() string

	// the current value of the target
	CurrentValue() interface{}

	// format an arbitrary value using suitable formatting method of the target
	FormatValue(val interface{}) string
}

// genericTarget is a way of encapsulating values that otherwise do not satisfy
// the target interface. useful when it is inconvient to give a value its own
// type
type genericTarget struct {
	label        string
	currentValue interface{}
}

func (trg genericTarget) Label() string {
	return trg.label
}

func (trg genericTarget) CurrentValue() interface{} {
	switch v := trg.currentValue.(type) {
	case func() interface{}:
		switch v := v().(type) {
		case error:
			panic(v)
		}
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
		switch v := t().(type) {
		case string:
			return val.(string)
		case error:
			panic(v)
		default:
			return fmt.Sprintf("%#v", val)
		}
	default:
		return fmt.Sprintf("%#v", val)
	}
}

// parseTarget uses a keyword to decide which part of the vcs to target
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
