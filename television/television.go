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
	VSync      bool
	VBlank     bool
	FrontPorch bool
	HSync      bool
	CBurst     bool
	Pixel      ColorSignal

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
	NewFrame() error
	NewScanline() error
	SetPixel(x, y int32, red, green, blue byte, vblank bool) error
	SetAltPixel(x, y int32, red, green, blue byte, vblank bool) error
}
