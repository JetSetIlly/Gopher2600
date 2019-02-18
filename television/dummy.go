package television

import "gopher2600/errors"

// DummyTV is the null implementation of the television interface. useful
// for tools that don't need a television or related information at all.
type DummyTV struct{ Television }

// NewDummyTV is the preferred method of initialisation for DummyTV - you can
// get away with an plain new(DummyTV) but this is probably more convenient
func NewDummyTV(tvType string, scale float32) (*DummyTV, error) {
	return new(DummyTV), nil
}

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

// Reset all the values for the television
func (DummyTV) Reset() error {
	return nil
}

// Signal (with DummyTV reciever) is the null implementation
func (DummyTV) Signal(SignalAttributes) error {
	return nil
}

// RequestTVState (with dummyTV reciever) is the null implementation
func (DummyTV) RequestTVState(request StateReq) (interface{}, error) {
	return nil, errors.NewGopherError(errors.UnknownTVRequest, request)
}

// RequestTVInfo (with dummyTV reciever) is the null implementation
func (DummyTV) RequestTVInfo(request MetaStateReq) (string, error) {
	return "", errors.NewGopherError(errors.UnknownTVRequest, request)
}

// RequestCallbackRegistration (with dummyTV reciever) is the null implementation
func (DummyTV) RequestCallbackRegistration(request CallbackReq, channel chan func(), callback func()) error {
	return errors.NewGopherError(errors.UnknownTVRequest, request)
}

// RequestSetAttr (with dummyTV reciever) is the null implementation
func (DummyTV) RequestSetAttr(request FeatureReq, args ...interface{}) error {
	return errors.NewGopherError(errors.UnknownTVRequest, request)
}
