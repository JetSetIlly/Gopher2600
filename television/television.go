package television

import "gopher2600/debugger/monitor"

// StateReq is used to identify which television attribute is being asked
// with the GetState() function
type StateReq int

// MetaStateReq is used to identify what information is being requested with the
// with the GetMetaState() function
type MetaStateReq int

// CallbackReq is used to identify which callback to register
type CallbackReq int

// FeatureReq is used to request the setting of a television attribute
// eg. setting debugging overscan
type FeatureReq int

// list of valid state requests
const (
	ReqFramenum StateReq = iota
	ReqScanline
	ReqHorizPos
)

// list of valid metastate requests
const (
	ReqTVSpec MetaStateReq = iota
	ReqLastKeyboard
	ReqLastMouse
	ReqLastMouseHorizPos
	ReqLastMouseScanline
)

// list of valid callback requests
const (
	ReqOnWindowClose CallbackReq = iota
	ReqOnKeyboard
	ReqOnMouseButtonLeft
	ReqOnMouseButtonRight
)

// list of valid feature requests
const (
	ReqSetVisibility         FeatureReq = iota // bool, optional bool (update on show)
	ReqSetVisibilityStable                     // none
	ReqSetAllowDebugging                       // bool
	ReqSetPause                                // bool
	ReqSetMasking                              // bool
	ReqToggleMasking                           // none
	ReqSetAltColors                            // bool
	ReqToggleAltColors                         // none
	ReqSetShowSystemState                      // bool
	ReqToggleShowSystemState                   // none
	ReqSetScale                                // float
	ReqIncScale                                // none
	ReqDecScale                                // none
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

// Television defines the operations that can be performed on the television
type Television interface {
	MachineInfoTerse() string
	MachineInfo() string

	Reset() error
	Signal(SignalAttributes) error

	GetState(StateReq) (interface{}, error)
	GetMetaState(MetaStateReq) (string, error)
	RegisterCallback(CallbackReq, chan func(), func()) error
	SetFeature(request FeatureReq, args ...interface{}) error

	SystemStateRecord(monitor.SystemState) error
}
