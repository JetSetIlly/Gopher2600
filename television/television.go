package television

import "gopher2600/hardware/tia/audio"

// Television defines the operations that can be performed on the conceptual
// television.  Implementations should not actually present the information,
// either visually or sonically. Instead, Renderers and Mixers can be added to
// perform those tasks.
//
// Note that for convenience, many television implementations "double-up" as
// Renderer interfaces. In these instances, the television will call
// AddRenderer() with itself as an argument.
type Television interface {
	String() string
	AddPixelRenderer(PixelRenderer)
	AddAudioMixer(AudioMixer)
	Reset() error
	Signal(SignalAttributes) error

	// Returns the value of the requested state. eg. the current scanline.
	GetState(StateReq) (int, error)

	// Returns the current specification the television is operating under
	GetSpec() *Specification
}

// PixelRenderer implementations displays, or otherwise works with, visal
// information from a television
//
// examples of renderers that display visual information:
//	* SDL/PixelTV
//	* ImageTV
//
// examples of renderers that do not display visual information but only work
// with it:
//	* DigestTV
//
// PixelRenderer implementations find it convenient to maintain a reference to
// the parent Television implementation and maybe even embed the Television
// interface. ie.
//
// type ExampleTV struct {
//		television.Television
//
//		...
// }
type PixelRenderer interface {
	NewFrame(frameNum int) error
	NewScanline(scanline int) error

	// setPixel() and setAltPixel() are called every cycle regardless of the
	// state of VBLANK and HBLANK.
	//
	// things to consider:
	//
	// o the x argument is measured from zero so renderers should decide how to
	//	handle pixels of during the HBLANK (x < ClocksPerHBLANK)
	//
	// o the y argument is also measure from zero but because VBLANK can be
	//	turned on at any time there's no easy test. the VBLANK flag is sent to
	//	help renderers decide what to do.
	//
	// o for renderers that are producing an accurate visual image, the pixel
	//	should always be set to video black if VBLANK is on.
	//
	//	some renderers however, may find it useful to set the pixel to the RGB
	//	value regardless of VBLANK. for example, DigestTV does this.
	//
	//	a vey important note is that some ROMs use VBLANK to control pixel
	//	color within the visible display area. ROMs affected:
	//
	//	* Custer's Revenge
	//	* Ladybug
	//
	SetPixel(x, y int, red, green, blue byte, vblank bool) error
	SetAltPixel(x, y int, red, green, blue byte, vblank bool) error

	// ChangeTVSpec is called when the television implementation decides to
	// change which TV specification is being used. Renderer implementations
	// should make sure that any data structures that depend on the
	// specification being used are still adequate.
	ChangeTVSpec() error
}

// AudioMixer implementations work with sound; most probably playing it.
type AudioMixer interface {
	SetAudio(audio audio.Audio) error
}

// SignalAttributes represents the data sent to the television
type SignalAttributes struct {
	VSync  bool
	VBlank bool
	CBurst bool
	HSync  bool
	Pixel  ColorSignal

	// AltPixel allows the emulator to set an alternative color for each pixel
	// - used to signal the debug color in addition to the regular color
	// - arguable that this be sent as some sort of meta-signal
	AltPixel ColorSignal

	// the HSyncSimple attribute is not part of the real TV spec. The signal
	// for a real flyback is the HSync signal (held for 8 color clocks).
	// however, this results in a confusing way of counting pixels - confusing
	// at least to people who are used to the Stella method of counting.
	//
	// if we were to use HSync to detect a new scanline then we have to treat
	// the front porch and back porch separately.  the convenient HSyncSimple
	// attribute effectively pushes the front and back porches together meaning
	// we can count from -68 to 159 - the same as Stella. this is helpful when
	// A/B testing.
	//
	// the TIA emulation sends both HSync and HSyncSimple signals.  television
	// implementations can use either, it doesn't really make any difference
	// except to debugging information. the "basic" television implementation
	// uses HSyncSimple instead of HSync
	HSyncSimple bool

	// audio signal is just the content of the VCS audio registers. for now,
	// sounds is generated/mixed by the television or gui implementation
	Audio       audio.Audio
	UpdateAudio bool
}

// StateReq is used to identify which television attribute is being asked
// with the GetState() function
type StateReq int

// list of valid state requests
const (
	ReqFramenum StateReq = iota
	ReqScanline
	ReqHorizPos
	ReqVisibleTop
	ReqVisibleBottom
)
