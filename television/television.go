package television

import (
	"fmt"
)

// Television defines the operations that can be performed on the television
type Television interface {
	Signal(vsync, vblank, frontPorch, hsync, cburst bool, pixel PixelSignal)
	MachineInfoTerse() string
	MachineInfo() string
	GetTVState(string) (*TVState, error)
	SetVisibility(visible bool) error
	SetPause(pause bool) error
}

// DummyTV is the null implementation of the television interface. useful
// for tools that don't need a television or related information at all.
type DummyTV struct{ Television }

// Signal (with DummyTV reciever) is the null implementation
func (DummyTV) Signal(vsync, vblank, frontPorch, hsync, cburst bool, pixel PixelSignal) {}

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

// GetTVState (with dummyTV reciever) is the null implementation
func (DummyTV) GetTVState(state string) (*TVState, error) {
	return nil, fmt.Errorf("dummy tv doesn't have that tv state (%s)", state)
}

// SetVisibility (with dummyTV reciever) is the null implementation
func (DummyTV) SetVisibility(visible bool) error {
	return nil
}

// SetPause (with dummyTV reciever) is the null implementation
func (DummyTV) SetPause(pause bool) error {
	return nil
}
