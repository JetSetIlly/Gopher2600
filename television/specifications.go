package television

// Specification is used to define the two television specifications
type Specification struct {
	ID     string
	Colors colors

	// the number of scanlines the 2600 Programmer's guide recommends for the
	// top/bottom parts of the screen:
	//
	// "A typical frame will consists of 3 vertical sync (VSYNC) lines*, 37 vertical
	// blank (VBLANK) lines, 192 TV picture lines, and 30 overscan lines. Atari’s
	// research has shown that this pattern will work on all types of TV sets."
	//
	// the above figures are in reference to the NTSC protocol
	scanlinesVSync    int
	scanlinesVBlank   int
	ScanlinesVisible  int
	scanlinesOverscan int

	// the total number of scanlines for the entire frame is the sum of the
	// four individual portions
	ScanlinesTotal int

	// the scanline at which the VBLANK should be turned off (Top) and
	// turned back on again (Bottom). the period between the top and bottom
	// scanline is the visible portion of the screen.
	//
	// in practice, the VCS can turn VBLANK on and off at any time; what the
	// two values below represent what "Atari's research" has shown to be safe.
	// by definition this means that:
	//
	//	Top = VSync + Vblank
	//
	//	Bottom = Top + Visible
	//
	// or
	//
	//	Bottom = Total - Overscan
	ScanlineTop    int
	ScanlineBottom int

	// the number of frames per second required by the specification
	FramesPerSecond int

	// AspectBias transforms the scaling factor for the X axis. in other words,
	// for width of every pixel is height of every pixel multiplied by the
	// aspect bias

	// AaspectBias transforms the scaling factor for the X axis.
	// values taken from Stella emualtor. useful for A/B testing
	AspectBias float32
}

// "Each scan lines starts with 68 clock counts of horizontal blank (not seen on
// the TV screen) followed by 160 clock counts to fully scan one line of TV
// picture. When the electron beam reaches the end of a scan line, it returns
// to the left side of the screen, waits for the 68 horizontal blank clock
// counts, and proceeds to draw the next line below."
//
// Horizontal clock counts are the same for both TV specificationst
const (
	HorizClksHBlank   = 68
	HorizClksVisible  = 160
	HorizClksScanline = 228
)

// SpecNTSC is the specification for NTSC television types
var SpecNTSC *Specification

// SpecPAL is the specification for PAL television types
var SpecPAL *Specification

func init() {
	SpecNTSC = &Specification{
		ID:                "NTSC",
		Colors:            colorsNTSC,
		scanlinesVSync:    3,
		scanlinesVBlank:   37,
		ScanlinesVisible:  192,
		scanlinesOverscan: 30,
		ScanlinesTotal:    262,
		FramesPerSecond:   60,
		AspectBias:        0.91,
	}

	SpecNTSC.ScanlineTop = SpecNTSC.scanlinesVBlank + SpecNTSC.scanlinesVSync
	SpecNTSC.ScanlineBottom = SpecNTSC.ScanlinesTotal - SpecNTSC.scanlinesOverscan

	SpecPAL = &Specification{
		ID:                "PAL",
		Colors:            colorsPAL,
		scanlinesVSync:    3,
		scanlinesVBlank:   45,
		ScanlinesVisible:  228,
		scanlinesOverscan: 36,
		ScanlinesTotal:    312,
		FramesPerSecond:   50,
		AspectBias:        1.09,
	}

	SpecPAL.ScanlineTop = SpecPAL.scanlinesVBlank + SpecPAL.scanlinesVSync
	SpecPAL.ScanlineBottom = SpecPAL.ScanlinesTotal - SpecPAL.scanlinesOverscan
}
