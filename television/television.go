package television

// Television represents the television. not part of the VCS but attached to it
type Television interface {
	Signal(vsync, vblank, frontPorch, hsync, cburst bool, color int)
	StringTerse() string
	String() string
}

// DummyTV is the null implementation of the television interface. useful
// for tools that don't need a television or related information at all.
type DummyTV struct{}

// Signal (with DummyTV reciever) is the minimalist implementation for the
// television interface
func (tv *DummyTV) Signal(vsync, vblank, frontPorch, hsync, cburst bool, color int) {}

// StringTerse returns the television information in terse format
func (tv DummyTV) StringTerse() string {
	return ""
}

// String returns the television information in verbose format
func (tv DummyTV) String() string {
	return ""
}
