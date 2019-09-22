package audio

import "gopher2600/hardware/memory"

// Audio contains all the components of the audio sub-system of the VCS TIA chip
type Audio struct {
	Control0 uint8
	Control1 uint8
	Freq0    uint8
	Freq1    uint8
	Volume0  uint8
	Volume1  uint8
}

// NewAudio is the preferred method of initialisation for the Video structure
func NewAudio() *Audio {
	au := new(Audio)
	return au
}

// AlterState checks the TIA memory for changes to registers that are
// interesting to the audio sub-system
func (au *Audio) AlterState(data memory.ChipData) {
	switch data.Name {
	case "AUDC0":
		au.Control0 = data.Value & 0x0f
	case "AUDC1":
		au.Control1 = data.Value & 0x0f
	case "AUDF0":
		au.Freq0 = data.Value & 0x1f
	case "AUDF1":
		au.Freq1 = data.Value & 0x1f
	case "AUDV0":
		au.Volume0 = data.Value & 0x0f
	case "AUDV1":
		au.Volume1 = data.Value & 0x0f
	}
}
