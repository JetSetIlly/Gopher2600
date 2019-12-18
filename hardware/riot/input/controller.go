package input

// Controller implementations feed controller Events to the device on
// request with the CheckInput() function. It maybe more convenient to use the
// device Handle() function directly.
type Controller interface {
	CheckInput(id ID) (Event, error)
}

// EventRecorder implementations make a record of events sent to the peripheral
// to which it is attached.
//
// Implementations should be able to handle being attached to more than one
// peripheral at once. The ID parameter of the EventRecord() function will help
// to differentiate between multiple devices.
//
// The implementation is free to use the EventRecord() event however it wants. A
// good example would be to match the event up with TV state information.
// Events can then be played back when the TV state matches the recorded state.
type EventRecorder interface {
	RecordEvent(id ID, event Event) error
}
