package television

import "gopher2600/errors"

// SignalAttributes represents the data sent to the television
type SignalAttributes struct {
	VSync, VBlank, FrontPorch, HSync, CBurst bool
	Pixel                                    PixelSignal
}

// TVStateReq is used to identify which television attribute is being asked
// for with the GetTVState() function
type TVStateReq string

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
	RegisterCallback(CallbackReq, func()) error
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
	return nil, errors.GopherError{Errno: errors.UnknownStateRequest, Values: errors.Values{request}}
}

// RegisterCallback (with dummyTV reciever) is the null implementation
func (DummyTV) RegisterCallback(request CallbackReq, callback func()) error {
	return errors.GopherError{Errno: errors.UnknownCallbackRequest, Values: errors.Values{request}}
}
