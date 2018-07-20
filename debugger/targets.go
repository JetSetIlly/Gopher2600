package debugger

import (
	"fmt"
	"gopher2600/hardware"
	"gopher2600/television"
)

// defines which types are valid targets
type target interface {
	Label() string
	ShortLabel() string
	ToInt() int
}

// parseTarget uses a keyword to decide which part of the vcs to target
func parseTarget(vcs *hardware.VCS, keyword string) (target, error) {
	var trg target
	var err error

	switch keyword {
	case "PC":
		trg = vcs.MC.PC
	case "A":
		trg = vcs.MC.A
	case "X":
		trg = vcs.MC.X
	case "Y":
		trg = vcs.MC.Y
	case "SP":
		trg = vcs.MC.SP
	case "FRAMENUM", "FRAME", "FR":
		trg, err = vcs.TV.RequestTVState(television.ReqFramenum)
	case "SCANLINE", "SL":
		trg, err = vcs.TV.RequestTVState(television.ReqScanline)
	case "HORIZPOS", "HP":
		trg, err = vcs.TV.RequestTVState(television.ReqHorizPos)
	default:
		return nil, fmt.Errorf("invalid target (%s)", keyword)
	}

	if err != nil {
		return nil, err
	}

	return trg, nil
}
