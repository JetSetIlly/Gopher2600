package debugger

import "gopher2600/hardware"

// defines which types are valid targets
type target interface {
	Label() string
	ShortLabel() string
	ToInt() int
}

// parseTarget uses a keyword to decide which part of the vcs to target
func parseTarget(vcs *hardware.VCS, keyword string) target {
	var target target
	var err error

	switch keyword {
	case "PC":
		target = vcs.MC.PC
	case "A":
		target = vcs.MC.A
	case "X":
		target = vcs.MC.X
	case "Y":
		target = vcs.MC.Y
	case "SP":
		target = vcs.MC.SP
	case "FRAMENUM", "FRAME", "FR":
		target, err = vcs.TV.GetTVState("FRAMENUM")
		if err != nil {
			return nil
		}
	case "SCANLINE", "SL":
		target, err = vcs.TV.GetTVState("SCANLINE")
		if err != nil {
			return nil
		}
	case "HORIZPOS", "HP":
		target, err = vcs.TV.GetTVState("HORIZPOS")
		if err != nil {
			return nil
		}
	}

	return target
}
