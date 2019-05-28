package audio

// Audio contains all the components of the audio sub-system of the VCS TIA chip
type Audio struct {
	control0 uint8
	control1 uint8
	freq0    uint8
	freq1    uint8
	volume0  uint8
	volume1  uint8
}

// NewAudio is the preferred method of initialisation for the Video structure
func NewAudio() *Audio {
	au := new(Audio)
	return au
}

// ReadMemory checks the TIA memory for changes to registers that are
// interesting to the audio sub-system
func (au *Audio) ReadMemory(register string, value uint8) bool {
	switch register {
	default:
		return false
	case "AUDC0":
		au.control0 = value & 0x0f
	case "AUDC1":
		au.control1 = value & 0x0f
	case "AUDF0":
		au.freq0 = value & 0x1f
	case "AUDF1":
		au.freq1 = value & 0x1f
	case "AUDV0":
		au.volume0 = value & 0x0f
	case "AUDV1":
		au.volume1 = value & 0x0f
	}

	return true
}
