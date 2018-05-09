package television

type specification struct {
	clocksPerHblank   int
	clocksPerScanline int

	vsyncClocks int

	scanlinesPerVBlank   int
	scanlinesPerVisible  int
	scanlinesPerOverscan int
	scanlinesTotal       int
}

var specNTSC *specification
var specPAL *specification

func init() {
	specNTSC = new(specification)
	if specNTSC == nil {
		panic("error during initialisation of NTSC specification")
	}
	specNTSC.clocksPerHblank = 68
	specNTSC.clocksPerScanline = 228
	specNTSC.vsyncClocks = 3
	specNTSC.scanlinesPerVBlank = 37
	specNTSC.scanlinesPerVisible = 228
	specNTSC.scanlinesPerOverscan = 30
	specNTSC.scanlinesTotal = 298

	specPAL = new(specification)
	if specPAL == nil {
		panic("error during initialisation of NTSC specification")
	}
	specPAL.clocksPerHblank = 68
	specPAL.clocksPerScanline = 228
	specPAL.vsyncClocks = 3
	specPAL.scanlinesPerVBlank = 45
	specPAL.scanlinesPerVisible = 228
	specPAL.scanlinesPerOverscan = 36
	specPAL.scanlinesTotal = 312
}
