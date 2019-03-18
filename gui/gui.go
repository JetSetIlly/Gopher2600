package gui

import "gopher2600/television"

// FeatureReq is used to request the setting of a gui attribute
// eg. setting debugging overscan
type FeatureReq int

// list of valid feature requests
const (
	ReqSetVisibility         FeatureReq = iota // bool, optional bool (update on show)
	ReqSetVisibilityStable                     // none
	ReqSetAllowDebugging                       // bool
	ReqSetPause                                // bool
	ReqSetMasking                              // bool
	ReqToggleMasking                           // none
	ReqSetAltColors                            // bool
	ReqToggleAltColors                         // none
	ReqSetShowSystemState                      // bool
	ReqToggleShowSystemState                   // none
	ReqSetScale                                // float
	ReqIncScale                                // none
	ReqDecScale                                // none
)

// GUI defines the operations that can be performed on GUIs
type GUI interface {
	television.Television
	SetFeature(request FeatureReq, args ...interface{}) error
	SetEventChannel(chan (Event))
}

// EventID idintifies the type of event taking place
type EventID int

// list of valid events
const (
	EventWindowClose EventID = iota
	EventKeyboard
	EventMouseLeft
	EventMouseRight
)

// EventData represents the data that is associated with an event
type EventData interface{}

// Event is the structure that is passed over the event channel
type Event struct {
	ID   EventID
	Data EventData
}

// EventDataKeyboard is the data that accompanies EvenKeyboard events
type EventDataKeyboard struct {
	Key  string
	Down bool
}

// EventDataMouse is the data that accompanies EventMouse events
type EventDataMouse struct {
	Down     bool
	X        int
	Y        int
	HorizPos int
	Scanline int
}
