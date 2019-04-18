package peripherals

// Transcriber implementations make a record of controller input
//
// The implementation is free to associate transciption information how it
// wants. a good example would be matching it up with free/scanline/horizpos
// and recording that information, along with the event information.  the
// recording can then be used as the source for controller input.
type Transcriber interface {
	Transcribe(id string, event Event) error
}
