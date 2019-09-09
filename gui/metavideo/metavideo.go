package metavideo

// Renderer implementations will add signal information to a presentation layer
// somehow.
type Renderer interface {
	MetaSignal(MetaSignalAttributes) error
}

// MetaSignalAttributes contains information about the last television signal. it is up to
// the Renderer to match this up with the last television signal
type MetaSignalAttributes struct {
	Label string

	// Renderer implementations are free to use the color information
	// as they wish (adding alpha information seems a probable scenario).
	Red, Green, Blue, Alpha byte

	// whether the meta-signal is one that is "instant" or resolves after a
	// short scheduled delay
	Scheduled bool
}
