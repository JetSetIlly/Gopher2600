package debugger

import (
	"fmt"
	"gopher2600/debugger/input"
	"gopher2600/errors"
	"gopher2600/television"
	"strings"
)

// defines which types are valid targets
type target interface {
	Label() string
	ShortLabel() string
	Value() interface{}
	FormatValue(interface{}) string
}

type genericTarget struct {
	label      string
	shortLabel string
	value      interface{}
}

func (trg genericTarget) Label() string {
	return trg.label
}

func (trg genericTarget) ShortLabel() string {
	return trg.shortLabel
}

func (trg genericTarget) Value() interface{} {
	switch v := trg.value.(type) {
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

func (trg genericTarget) FormatValue(fv interface{}) string {
	switch v := trg.value.(type) {
	case string:
		return fv.(string)
	case func() interface{}:
		switch v := v().(type) {
		case string:
			return fv.(string)
		case error:
			panic(v)
		default:
			return fmt.Sprintf("%v", fv)
		}
	default:
		return fmt.Sprintf("%v", fv)
	}
}

// parseTarget uses a keyword to decide which part of the vcs to target
func parseTarget(dbg *Debugger, tokens *input.Tokens) (target, error) {
	var trg target
	var err error

	keyword, present := tokens.Get()
	if present {
		keyword = strings.ToUpper(keyword)
		switch keyword {
		// cpu registers
		case "PC":
			trg = dbg.vcs.MC.PC
		case "A":
			trg = dbg.vcs.MC.A
		case "X":
			trg = dbg.vcs.MC.X
		case "Y":
			trg = dbg.vcs.MC.Y
		case "SP":
			trg = dbg.vcs.MC.SP

		// tv state
		case "FRAMENUM", "FRAME", "FR":
			trg = &genericTarget{
				label:      "Frame",
				shortLabel: "FR",
				value: func() interface{} {
					if dbg.lastResult == nil {
						return -1
					}
					fr, err := dbg.vcs.TV.GetState(television.ReqFramenum)
					if err != nil {
						return err
					}
					return fr.(int)
				},
			}
		case "SCANLINE", "SL":
			trg = &genericTarget{
				label:      "Scanline",
				shortLabel: "SL",
				value: func() interface{} {
					if dbg.lastResult == nil {
						return -1
					}
					sl, err := dbg.vcs.TV.GetState(television.ReqScanline)
					if err != nil {
						return err
					}
					return sl.(int)
				},
			}
		case "HORIZPOS", "HP":
			trg = &genericTarget{
				label:      "Horiz Pos",
				shortLabel: "HP",
				value: func() interface{} {
					if dbg.lastResult == nil {
						return -1
					}
					hp, err := dbg.vcs.TV.GetState(television.ReqHorizPos)
					if err != nil {
						return err
					}
					return hp.(int)
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
						label:      "INSTRUCTION EFFECT",
						shortLabel: "INS EFF",
						value: func() interface{} {
							if dbg.lastResult == nil {
								return -1
							}
							return int(dbg.lastResult.Defn.Effect)
						},
					}
				default:
					err = errors.NewGopherError(errors.InvalidTarget, fmt.Sprintf("%s/%s", keyword, subkey))
				}
			} else {
				err = errors.NewGopherError(errors.InvalidTarget, keyword)
			}

		// cartridge
		case "CARTRIDGE", "CART":
			subkey, present := tokens.Get()
			if present {
				subkey = strings.ToUpper(subkey)
				switch subkey {
				case "BANK":
					trg = &genericTarget{
						label:      "BANK",
						shortLabel: "BANK",
						value: func() interface{} {
							return dbg.vcs.Mem.Cart.Bank
						},
					}
				default:
					err = errors.NewGopherError(errors.InvalidTarget, fmt.Sprintf("%s/%s", keyword, subkey))
				}
			} else {
				err = errors.NewGopherError(errors.InvalidTarget, keyword)
			}

		default:
			err = errors.NewGopherError(errors.InvalidTarget, keyword)
		}
	}

	return trg, err
}
