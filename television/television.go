package television

import "gopher2600/errors"

// list of valid requests for television implementations. it is not
// required that every implementation does something useful for every request.
// for instance, ONWINDOWCLOSE is meaningless if the implementation has no
// display window
const (
	ReqFramenum TVStateReq = "FRAME"
	ReqScanline TVStateReq = "SCANLINE"
	ReqHorizPos TVStateReq = "HORIZPOS"

	ReqTVSpec     TVInfoReq = "TVSPEC"
	ReqLastMouse  TVInfoReq = "MOUSE"
	ReqLastMouseX TVInfoReq = "MOUSEX"
	ReqLastMouseY TVInfoReq = "MOUSEY"

	ReqOnWindowClose      CallbackReq = "ONWINDOWCLOSE"
	ReqOnMouseButtonLeft  CallbackReq = "ONMOUSEBUTTONLEFT"
	ReqOnMouseButtonRight CallbackReq = "ONMOUSEBUTTONRIGHT"
)

// SignalAttributes represents the data sent to the television
type SignalAttributes struct {
	VSync, VBlank, FrontPorch, HSync, CBurst bool
	Pixel                                    PixelSignal
}

// TVStateReq is used to identify which television attribute is being asked
// for with the GetTVState() function
type TVStateReq string

// TVInfoReq is used to identiry what information is being requested with the
// GetTVInfo() function
type TVInfoReq string

// CallbackReq is used to identify which callback to register
type CallbackReq string

// Television defines the operations that can be performed on the television
type Television interface {
	MachineInfoTerse() string
	MachineInfo() string
	Signal(SignalAttributes)
	SetVisibility(visible bool) error
	SetPause(pause bool) error

	RequestTVState(TVStateReq) (*TVState, error)
	RequestTVInfo(TVInfoReq) (string, error)
	RegisterCallback(CallbackReq, chan func(), func()) error
}

// DummyTV is the null implementation of the television interface. useful
// for tools that don't need a television or related information at all.
type DummyTV struct{ Television }

// MachineInfoTerse (with DummyTV reciever) is the null implementation
func (DummyTV) MachineInfoTerse() string {
	return ""
}

// MachineInfo (with DummyTV reciever) is the null implementation
func (DummyTV) MachineInfo() string {
	return ""
}

// map String to MachineInfo
func (tv DummyTV) String() string {
	return tv.MachineInfo()
}

// Signal (with DummyTV reciever) is the null implementation
func (DummyTV) Signal(SignalAttributes) {}

// SetVisibility (with dummyTV reciever) is the null implementation
func (DummyTV) SetVisibility(visible bool) error {
	return nil
}

// SetPause (with dummyTV reciever) is the null implementation
func (DummyTV) SetPause(pause bool) error {
	return nil
}

// RequestTVState (with dummyTV reciever) is the null implementation
func (DummyTV) RequestTVState(request TVStateReq) (*TVState, error) {
	return nil, errors.NewGopherError(errors.UnknownTVRequest, request)
}

// RequestTVInfo (with dummyTV reciever) is the null implementation
func (DummyTV) RequestTVInfo(request TVInfoReq) (string, error) {
	return "", errors.NewGopherError(errors.UnknownTVRequest, request)
}

// RegisterCallback (with dummyTV reciever) is the null implementation
func (DummyTV) RegisterCallback(request CallbackReq, channel chan func(), callback func()) error {
	return errors.NewGopherError(errors.UnknownTVRequest, request)
}
