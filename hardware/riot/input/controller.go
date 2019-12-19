package input

// Controller implementations feed controller Events to the device on
// request with the CheckInput() function. It maybe more convenient to use the
// device Handle() function directly.
type Controller interface {
	CheckInput(id ID) (Event, error)
}

// EventRecorder implementations mirror an incoming event. Originally intended
// to mirror the event to a file on disk but it could be for any purpose I
// suppose.
//
// Implementations should be able to handle being attached to more than one
// peripheral at once. The ID parameter of the EventRecord() function will help
// to differentiate between multiple devices.
type EventRecorder interface {
	RecordEvent(id ID, event Event) error
}
