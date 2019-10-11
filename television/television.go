package television

import "gopher2600/hardware/tia/audio"

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
	Audio audio.Audio
}

// Television defines the operations that can be performed on television
// implementations
type Television interface {
	String() string

	AddRenderer(Renderer)
	AddMixer(AudioMixer)

	Reset() error
	Signal(SignalAttributes) error

	GetState(StateReq) (int, error)
	GetSpec() *Specification
}

// Renderer implementations display information from a television
type Renderer interface {
	NewFrame(frameNum int) error
	NewScanline(scanline int) error
	SetPixel(x, y int32, red, green, blue byte, vblank bool) error
	SetAltPixel(x, y int32, red, green, blue byte, vblank bool) error

	// ChangeTVSpec is called when the television implementation decides to
	// change which TV specification is being used. Renderer implementations
	// should make sure that any data structures that depend on the
	// specification being used are still adequate.
	ChangeTVSpec() error
}

// AudioMixer implementations play sound
type AudioMixer interface {
	SetAudio(audio audio.Audio) error
}
