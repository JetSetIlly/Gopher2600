package television

import "gopher2600/debugger/metavideo"

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
	// - arguable that this be sent as a metasignal
	AltPixel ColorSignal
}

// Television defines the operations that can be performed on television
// implementations
type Television interface {
	MachineInfoTerse() string
	MachineInfo() string

	AddRenderer(Renderer)

	Reset() error
	Signal(SignalAttributes) error
	MetaSignal(metavideo.MetaSignalAttributes) error

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
