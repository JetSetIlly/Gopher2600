package television

// Specification is used to define the two television specifications
type Specification struct {
	ID string

	ClocksPerHblank   int
	ClocksPerVisible  int
	ClocksPerScanline int

	ScanlinesPerVSync    int
	ScanlinesPerVBlank   int
	ScanlinesPerVisible  int
	ScanlinesPerOverscan int
	ScanlinesTotal       int

	VsyncClocks int

	Colors []color

	IdealTop       int
	IdealBottom    int
	IdealScanlines int
}

// TranslateColorSignal decodaes color signal to an RGB value
func (spec Specification) TranslateColorSignal(sig ColorSignal) (byte, byte, byte) {
	red, green, blue := byte(0), byte(0), byte(0)
	if sig != VideoBlack {
		col := spec.Colors[sig]
		red, green, blue = byte((col&0xff0000)>>16), byte((col&0xff00)>>8), byte(col&0xff)
	}

	return red, green, blue
}

// SpecNTSC is the specification for NTSC television typee
var SpecNTSC *Specification

// SpecPAL is the specification for PAL television typee
var SpecPAL *Specification

func init() {
	SpecNTSC = new(Specification)
	SpecNTSC.ID = "NTSC"
	SpecNTSC.ClocksPerHblank = 68
	SpecNTSC.ClocksPerVisible = 160
	SpecNTSC.ClocksPerScanline = 228
	SpecNTSC.ScanlinesPerVSync = 3
	SpecNTSC.ScanlinesPerVBlank = 37
	SpecNTSC.ScanlinesPerVisible = 192
	SpecNTSC.ScanlinesPerOverscan = 30
	SpecNTSC.ScanlinesTotal = 262
	SpecNTSC.Colors = ntscColors
	SpecNTSC.VsyncClocks = SpecNTSC.ScanlinesPerVSync * SpecNTSC.ClocksPerScanline
	SpecNTSC.IdealTop = SpecNTSC.ScanlinesPerVSync + SpecNTSC.ScanlinesPerVBlank
	SpecNTSC.IdealBottom = SpecNTSC.ScanlinesTotal - SpecNTSC.ScanlinesPerOverscan
	SpecNTSC.IdealScanlines = SpecNTSC.IdealBottom - SpecNTSC.IdealTop

	SpecPAL = new(Specification)
	SpecPAL.ID = "PAL"
	SpecPAL.ClocksPerHblank = 68
	SpecPAL.ClocksPerVisible = 160
	SpecPAL.ClocksPerScanline = 228
	SpecPAL.ScanlinesPerVBlank = 45
	SpecPAL.ScanlinesPerVisible = 228
	SpecPAL.ScanlinesPerOverscan = 36
	SpecPAL.ScanlinesTotal = 312
	SpecPAL.VsyncClocks = SpecPAL.ScanlinesPerVSync * SpecPAL.ClocksPerScanline
	SpecPAL.IdealTop = SpecPAL.ScanlinesPerVSync + SpecPAL.ScanlinesPerVBlank
	SpecPAL.IdealBottom = SpecPAL.ScanlinesTotal - SpecPAL.ScanlinesPerOverscan
	SpecPAL.IdealScanlines = SpecPAL.IdealBottom - SpecPAL.IdealTop

	// use NTSC colors for PAL specification for now
	// TODO: implement PAL colors
	SpecPAL.Colors = ntscColors
}
