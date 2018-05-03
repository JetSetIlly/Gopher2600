package television

// Television represents the television. not part of the VCS but attached to it
type Television interface {
	Signal(vsync, vblank, frontPorch, hsync, cburst bool, color int)
}
