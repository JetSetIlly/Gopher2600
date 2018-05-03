package television

// DummyTV is the null implementation of the Television interface
type DummyTV struct{}

// Signal is how the VCS communicates with the televsion
func (tv *DummyTV) Signal(vsync, vblank, frontPorch, hsync, cburst bool, color int) {
}
