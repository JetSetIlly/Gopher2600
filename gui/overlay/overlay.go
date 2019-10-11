package overlay

// Renderer implementations will add signal information to a presentation layer
// somehow.
type Renderer interface {
	OverlaySignal(Signal) error
}

// Signal contains additional debugging information from the last video cycle.
// it is up to the Renderer to match this up with the last television signal
type Signal struct {
	Label string

	// Renderer implementations are free to use the color information
	// as they wish (adding alpha information seems a probable scenario).
	Red, Green, Blue, Alpha byte

	// whether the attribute is one that is "instant" or resolves after a
	// short scheduled delay
	Scheduled bool
}
