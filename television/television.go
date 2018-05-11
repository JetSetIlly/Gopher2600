package television

import (
	"fmt"
	"gopher2600/hardware/tia/video"
)

// Television defines the operations that can be performed on the television
type Television interface {
	Signal(vsync, vblank, frontPorch, hsync, cburst bool, color video.Color)
	StringTerse() string
	String() string
	GetTVState(string) (*TVState, error)
	ForceUpdate() error
	SetVisibility(visible bool) error
}

// DummyTV is the null implementation of the television interface. useful
// for tools that don't need a television or related information at all.
type DummyTV struct{ Television }

// Signal (with DummyTV reciever) is the null implementation
func (tv *DummyTV) Signal(vsync, vblank, frontPorch, hsync, cburst bool, color video.Color) {}

// StringTerse (with DummyTV reciever) is the null implementation
func (tv DummyTV) StringTerse() string {
	return ""
}

// String (with DummyTV reciever) is the null implementation
func (tv DummyTV) String() string {
	return ""
}

// GetTVState (with dummyTV reciever) is the null implementation
func (tv DummyTV) GetTVState(state string) (*TVState, error) {
	return nil, fmt.Errorf("dummy tv doesn't have that tv state (%s)", state)
}

// ForceUpdate (with dummyTV reciever) is the null implementation
func (tv DummyTV) ForceUpdate() error {
	return nil
}

// SetVisibility (with dummyTV reciever) is the null implementation
func (tv DummyTV) SetVisibility(visible bool) error {
	return nil
}
