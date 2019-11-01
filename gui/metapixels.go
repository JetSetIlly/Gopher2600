package gui

// MetaPixelRenderer implementations accepts MetaPixel values and associates it
// in some way with the moste recent television signal
type MetaPixelRenderer interface {
	SetMetaPixel(MetaPixel) error
}

// MetaPixel contains additional debugging information from the last video cycle.
// it is up to the Renderer to match this up with the last television signal
type MetaPixel struct {
	Label string

	// Renderer implementations are free to use the color information
	// as they wish (adding alpha information seems a probable scenario).
	Red, Green, Blue, Alpha byte

	// whether the attribute is one that is "instant" or resolves after a
	// short scheduled delay
	Scheduled bool
}
