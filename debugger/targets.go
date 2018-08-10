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
}

// genericTarget is a a way of hacking any object so that it qualifies as a
// target. bit messy but it may more convenient that defining the interface for
// a given type
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
	switch value := trg.value.(type) {
	case func() interface{}:
		return value()
	default:
		return value
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
		case "FRAMENUM", "FRAME", "FR":
			trg, err = dbg.vcs.TV.RequestTVState(television.ReqFramenum)
		case "SCANLINE", "SL":
			trg, err = dbg.vcs.TV.RequestTVState(television.ReqScanline)
		case "HORIZPOS", "HP":
			trg, err = dbg.vcs.TV.RequestTVState(television.ReqHorizPos)

		// cpu instruction targetting was originally added as an experiment, to
		// help investigate a bug in the emulation. I don't think it's much use
		// but it was an instructive exercise and may come in useful one day.
		case "INSTRUCTION", "INS":
			subkey, present := tokens.Get()
			if present {
				subkey = strings.ToUpper(subkey)
				switch subkey {
				case "EFFECT", "EFF":
					return &genericTarget{
						label:      "EFFECT",
						shortLabel: "EFF",
						value: func() interface{} {
							if dbg.lastResult == nil {
								return -1
							}
							return int(dbg.lastResult.Defn.Effect)
						},
					}, nil
				default:
					return nil, errors.NewGopherError(errors.InvalidTarget, fmt.Sprintf("%s/%s", keyword, subkey))
				}
			} else {
				return nil, errors.NewGopherError(errors.InvalidTarget, keyword)
			}
		default:
			return nil, errors.NewGopherError(errors.InvalidTarget, keyword)
		}
	}

	return trg, err
}
