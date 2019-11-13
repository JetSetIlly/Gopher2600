package peripherals

// Transcriber implementations make a record of events sent to the peripheral
// to which it is attached.
//
// implementations should be able to handle being attached to more than one
// peripheral at once. the id parameter will help with this
//
// The implementation is free to use the transcribed event how it wants. a good
// example would be matching it up with free/scanline/horizpos and recording
// that information, along with the event information. the transcription can then
// be used as the source for controller input (by implementing the Controller
// interface).
type Transcriber interface {
	Transcribe(id PeriphID, event Action) error
}
