package television

// Specification is used to define the two television specifications
type Specification struct {
	ID string

	ScanlinesPerVSync    int
	ScanlinesPerVBlank   int
	ScanlinesPerVisible  int
	ScanlinesPerOverscan int
	ScanlinesTotal       int

	Colors colors

	FramesPerSecond float64
	SecondsPerFrame float64

	// AspectBias transforms the scaling factor for the X axis.
	AspectBias float32
}

// ClocksPerHblank is the same for all tv specifications
const ClocksPerHblank = 68

// ClocksPerVisible is the same for all tv specifications
const ClocksPerVisible = 160

// ClocksPerScanline is the same for all tv specifications
const ClocksPerScanline = 228

// SpecNTSC is the specification for NTSC television typee
var SpecNTSC *Specification

// SpecPAL is the specification for PAL television typee
var SpecPAL *Specification

func init() {
	SpecNTSC = new(Specification)
	SpecNTSC.ID = "NTSC"
	SpecNTSC.ScanlinesPerVSync = 3
	SpecNTSC.ScanlinesPerVBlank = 37
	SpecNTSC.ScanlinesPerVisible = 192
	SpecNTSC.ScanlinesPerOverscan = 30
	SpecNTSC.ScanlinesTotal = 262
	SpecNTSC.FramesPerSecond = 60.0
	SpecNTSC.SecondsPerFrame = 1.0 / SpecNTSC.FramesPerSecond
	SpecNTSC.Colors = colorsNTSC

	SpecPAL = new(Specification)
	SpecPAL.ID = "PAL"
	SpecPAL.ScanlinesPerVSync = 3
	SpecPAL.ScanlinesPerVBlank = 45
	SpecPAL.ScanlinesPerVisible = 228
	SpecPAL.ScanlinesPerOverscan = 36
	SpecPAL.ScanlinesTotal = 312
	SpecPAL.FramesPerSecond = 50.0
	SpecPAL.SecondsPerFrame = 1.0 / SpecPAL.FramesPerSecond
	SpecPAL.Colors = colorsPAL

	// AaspectBias transforms the scaling factor for the X axis.
	// values taken from Stella emualtor. i've no idea from where these values
	// were originated but they're useful for A/B testing
	SpecNTSC.AspectBias = 0.91
	SpecPAL.AspectBias = 1.09
}
