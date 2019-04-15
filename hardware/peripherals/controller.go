package peripherals

// Controller defines the operations required for VCS controllers
type Controller interface {
	// make sure the most recent input is ready for the emulation
	Strobe() error

	// handle interprets the supplied action and updates the emulation
	Handle(action string) error

	// add InputTranscriber implementation for consideration by the controller
	RegisterTranscriber(Transcriber)
}

// Transcriber defines the operation required for a transcriber (observer) of
// VCS controller input
type Transcriber interface {
	Transcribe(action string)
}
