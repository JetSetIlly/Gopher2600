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

	// use NTSC colors for PAL specification for now
	// TODO: implement PAL colors
	SpecPAL.Colors = ntscColors
}
