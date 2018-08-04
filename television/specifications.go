package television

type specification struct {
	ClocksPerHblank   int
	ClocksPerVisible  int
	ClocksPerScanline int

	VsyncClocks int

	ScanlinesPerVBlank   int
	ScanlinesPerVisible  int
	ScanlinesPerOverscan int
	ScanlinesTotal       int

	Colors []color
}

var specNTSC *specification
var specPAL *specification

func init() {
	specNTSC = new(specification)
	specNTSC.ClocksPerHblank = 68
	specNTSC.ClocksPerVisible = 160
	specNTSC.ClocksPerScanline = 228
	specNTSC.VsyncClocks = 3 * specNTSC.ClocksPerScanline
	specNTSC.ScanlinesPerVBlank = 37
	specNTSC.ScanlinesPerVisible = 228
	specNTSC.ScanlinesPerOverscan = 30
	specNTSC.ScanlinesTotal = 298
	specNTSC.Colors = ntscColors

	specPAL = new(specification)
	specPAL.ClocksPerHblank = 68
	specPAL.ClocksPerVisible = 160
	specPAL.ClocksPerScanline = 228
	specPAL.VsyncClocks = 3 * specPAL.ClocksPerScanline
	specPAL.ScanlinesPerVBlank = 45
	specPAL.ScanlinesPerVisible = 228
	specPAL.ScanlinesPerOverscan = 36
	specPAL.ScanlinesTotal = 312

	// use NTSC colors for PAL specification for now
	// TODO: implement PAL colors
	specPAL.Colors = ntscColors
}
