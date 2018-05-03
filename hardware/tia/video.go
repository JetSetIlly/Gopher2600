package tia

// Video contains all the sub-components of the video component of the VCS TIA chip
type Video struct {
	tia *TIA
}

// NewVideo is the preferred method of initialisation for the Video structure
func NewVideo(tia *TIA) *Video {
	vd := new(Video)
	if vd == nil {
		return nil
	}
	vd.tia = tia
	return vd
}

func (vd *Video) serviceTIAMemory(register string, value uint8) bool {
	switch register {
	case "NUSIZ0":
	case "NUSIZ1":
	case "COLUP0":
	case "COLUP1":
	case "COLUPF":
	case "COLUBK":
	case "CTRLPF":
	case "REFP0":
	case "REFP1":
	case "PF0":
	case "PF1":
	case "PF2":
	case "RESP0":
	case "RESP1":
	case "RESM0":
	case "RESM1":
	case "RESBL":
	case "GRP0":
	case "GRP1":
	case "ENAM0":
	case "ENAM1":
	case "ENABL":
	case "HMP0":
	case "HMP1":
	case "HMM0":
	case "HMM1":
	case "HMBL":
	case "VDELP0":
	case "VDELP1":
	case "VDELBL":
	case "RESMP0":
	case "RESMP1":
	case "HMCLR":
	case "CXCLR":
	}

	return false
}
