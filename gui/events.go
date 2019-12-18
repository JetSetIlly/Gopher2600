package gui

// Events are the things that happen in the gui, as a result of user interaction,
// and sent over a registered event channel.

// EventID idintifies the type of event taking place
type EventID int

// list of valid events
const (
	EventWindowClose EventID = iota
	EventKeyboard
	EventMouseLeft
	EventMouseRight
)

// KeyMod identifies
type KeyMod int

// list of valud key modifiers
const (
	KeyModNone KeyMod = iota
	KeyModShift
	KeyModCtrl
	KeyModAlt
)

// EventData represents the data that is associated with an event
type EventData interface{}

// Event is the structure that is passed over the event channel
//
// Do not confuse this with the peripheral Event type.
type Event struct {
	ID   EventID
	Data EventData
}

// EventDataKeyboard is the data that accompanies EvenKeyboard events
type EventDataKeyboard struct {
	Key  string
	Down bool
	Mod  KeyMod
}

// EventDataMouse is the data that accompanies EventMouse events
type EventDataMouse struct {
	Down     bool
	X        int
	Y        int
	HorizPos int
	Scanline int
}
