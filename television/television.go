package television

import "gopher2600/debugger/monitor"

// StateReq is used to identify which television attribute is being asked
// with the GetState() function
type StateReq int

// list of valid state requests
const (
	ReqFramenum StateReq = iota
	ReqScanline
	ReqHorizPos
	ReqTVSpec
)

// SignalAttributes represents the data sent to the television
type SignalAttributes struct {
	VSync, VBlank, FrontPorch, HSync, CBurst bool
	Pixel                                    ColorSignal

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

	Reset() error
	Signal(SignalAttributes) error

	GetState(StateReq) (interface{}, error)
	SystemStateRecord(monitor.SystemState) error
}
