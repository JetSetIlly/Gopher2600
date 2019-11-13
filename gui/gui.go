package gui

// GUI defines the operations that can be performed on visual user interfaces.
//
// Currently, GUI implementations expect also to be an instance of
// television.Television. This way a single object can be used in both GUI and
// television contexts. In practice, the GUI instance may also implement the
// Renderer and AudioMixer interfaces from the television packages but this is
// not mandated by the GUI interface.
type GUI interface {
	// All GUIs should implement a MetaPixelRenderer even if only a stub
	MetaPixelRenderer

	// returns true if GUI is currently visible. false if not
	IsVisible() bool

	// send a request to set a gui feature
	SetFeature(request FeatureReq, args ...interface{}) error

	// the event channel is used to by the GUI implementation to send
	// information back to the main program. the GUI may or may not be in its
	// own go routine but in regardless, the event channel is used for this
	// purpose.
	SetEventChannel(chan (Event))
}
