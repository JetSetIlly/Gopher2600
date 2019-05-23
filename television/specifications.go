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

	Colors colors

	FramesPerSecond float64
	SecondsPerFrame float64
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
	SpecNTSC.Colors = colorsNTSC
	SpecNTSC.FramesPerSecond = 60.0
	SpecNTSC.SecondsPerFrame = 1.0 / SpecNTSC.FramesPerSecond

	SpecPAL = new(Specification)
	SpecPAL.ID = "PAL"
	SpecPAL.ClocksPerHblank = 68
	SpecPAL.ClocksPerVisible = 160
	SpecPAL.ClocksPerScanline = 228
	SpecPAL.ScanlinesPerVSync = 3
	SpecPAL.ScanlinesPerVBlank = 45
	SpecPAL.ScanlinesPerVisible = 228
	SpecPAL.ScanlinesPerOverscan = 36
	SpecPAL.ScanlinesTotal = 312
	SpecPAL.FramesPerSecond = 50.0
	SpecPAL.SecondsPerFrame = 1.0 / SpecPAL.FramesPerSecond

	// use NTSC colors for PAL specification for now
	// TODO: implement PAL colors
	SpecPAL.Colors = colorsNTSC
}
